/**
 * MSE (Media Source Extensions) transport for the compressed player.
 *
 * Chromium/Firefox can play the server's chunked, non-seekable fragmented MP4
 * directly through `<video src>`. WebKit (iPadOS/iOS and desktop Safari)
 * requires HTTP byte ranges for a media source and refuses that stream, so the
 * exact same bytes are fetched and fed into a SourceBuffer instead.
 *
 * Nothing changes on the server: the endpoint still streams ffmpeg's stdout as
 * it is produced and never writes to disk.
 */

/**
 * Codec candidates, most specific first. The first must match the encoder pin
 * in `video_compress_serve.go` (`-profile:v high -level:v 4.2`), which yields
 * `avc1.64002a`. The rest are safety nets for older encoders/devices.
 */
const CODEC_CANDIDATES = [
  'video/mp4; codecs="avc1.64002a,mp4a.40.2"',
  'video/mp4; codecs="avc1.640029,mp4a.40.2"',
  'video/mp4; codecs="avc1.4d4029,mp4a.40.2"',
  'video/mp4; codecs="avc1.42e01e,mp4a.40.2"',
];

/** Buffered-ahead window before the fetch is paused. */
const HIGH_WATER_SECONDS = 30;
/** Played-out data kept behind the playhead for cheap rewind. */
const KEEP_BEHIND_SECONDS = 10;

/** Backing store is always a plain ArrayBuffer here (never SharedArrayBuffer). */
type Bytes = Uint8Array<ArrayBuffer>;

/** ManagedMediaSource is not in lib.dom yet; only Safari implements it. */
interface ManagedMediaSourceLike extends MediaSource {
  readonly streaming: boolean;
}

interface MediaSourceConstructor {
  new (): MediaSource;
  isTypeSupported(type: string): boolean;
}

interface ManagedMediaSourceConstructor {
  new (): ManagedMediaSourceLike;
  isTypeSupported(type: string): boolean;
}

/** Swallow a promise rejection we cannot act on (autoplay/abort races). */
const ignore = (): void => undefined;

function resolveMseConstructor(): {
  ctor: MediaSourceConstructor;
  managed: boolean;
} | null {
  const scope = globalThis as unknown as {
    ManagedMediaSource?: ManagedMediaSourceConstructor;
    MediaSource?: MediaSourceConstructor;
  };
  if (scope.ManagedMediaSource) {
    return { ctor: scope.ManagedMediaSource, managed: true };
  }
  if (scope.MediaSource) {
    return { ctor: scope.MediaSource, managed: false };
  }
  return null;
}

/** First codec the browser accepts, or null when MSE cannot play the stream. */
export function pickMseCodec(): string | null {
  const resolved = resolveMseConstructor();
  if (!resolved) return null;
  for (const codec of CODEC_CANDIDATES) {
    try {
      if (resolved.ctor.isTypeSupported(codec)) return codec;
    } catch {
      // isTypeSupported can throw on malformed strings; keep probing.
    }
  }
  return null;
}

/** True when the browser exposes an MSE implementation for our codec. */
export function isMseSupported(): boolean {
  return pickMseCodec() !== null;
}

function isIOS(): boolean {
  const ua = navigator.userAgent;
  if (/iPad|iPhone|iPod/.test(ua)) return true;
  // iPadOS 13+ masquerades as desktop Safari but reports touch points.
  return navigator.maxTouchPoints > 1 && ua.includes("Macintosh");
}

function isDesktopSafari(): boolean {
  if (isIOS()) return false;
  const ua = navigator.userAgent;
  return (
    ua.includes("Safari") &&
    !/Chrome|Chromium|CriOS|Edg|EdgiOS|OPR|Android/.test(ua)
  );
}

/**
 * True when the compressed stream must be delivered through MSE. Every WebKit
 * browser on an Apple platform requires range support, so native progressive
 * playback of the transcode is impossible there.
 */
export function shouldUseMse(): boolean {
  return (isIOS() || isDesktopSafari()) && isMseSupported();
}

/* -------------------- fragmented-MP4 splitting -------------------- */

/** Copy a streamed chunk into a plain ArrayBuffer-backed view. */
function toBytes(chunk: Uint8Array): Bytes {
  const copy = new Uint8Array(chunk.length);
  copy.set(chunk);
  return copy;
}

function concat(a: Bytes, b: Bytes): Bytes {
  if (a.length === 0) return b.slice();
  if (b.length === 0) return a;
  const out = new Uint8Array(a.length + b.length);
  out.set(a, 0);
  out.set(b, a.length);
  return out;
}

function join(parts: Bytes[]): Bytes {
  if (parts.length === 1) return parts[0];
  let total = 0;
  for (const part of parts) total += part.length;
  const out = new Uint8Array(total);
  let at = 0;
  for (const part of parts) {
    out.set(part, at);
    at += part.length;
  }
  return out;
}

/**
 * Splits a growing fragmented-MP4 byte stream into appendable units: one
 * initialization segment (`ftyp`+`moov`) followed by one unit per media
 * fragment (`styp`?+`moof`+`mdat`).
 *
 * MSE expects whole segments, but a TCP chunk boundary can land anywhere, so
 * appends are only emitted at top-level box edges. `-movflags` on the server
 * guarantees `mvex` in the init segment and a `tfdt` per fragment, which is
 * what the ISO-BMFF MSE byte stream format requires.
 */
class Fmp4Splitter {
  private pending: Bytes = new Uint8Array(0);
  private unit: Bytes[] = [];
  private hasMdat = false;

  push(chunk: Uint8Array): Bytes[] {
    this.pending = concat(this.pending, toBytes(chunk));
    const out: Bytes[] = [];
    const bytes = this.pending;
    let offset = 0;

    while (bytes.length - offset >= 8) {
      const view = new DataView(
        bytes.buffer,
        bytes.byteOffset + offset,
        bytes.length - offset
      );
      let size = view.getUint32(0);
      const type = String.fromCharCode(
        bytes[offset + 4],
        bytes[offset + 5],
        bytes[offset + 6],
        bytes[offset + 7]
      );
      if (size === 1) {
        if (bytes.length - offset < 16) break;
        size = view.getUint32(8) * 4294967296 + view.getUint32(12);
      } else if (size === 0) {
        // Box extends to end of stream; only meaningful for the last box.
        size = bytes.length - offset;
      }
      if (size < 8 || bytes.length - offset < size) break;

      const box = bytes.slice(offset, offset + size);
      offset += size;

      if (type === "ftyp" || type === "moov") {
        this.unit.push(box);
        // The init segment is complete once moov is in.
        if (type === "moov") this.emit(out);
        continue;
      }
      if (type === "styp" || type === "moof") {
        // A new fragment begins: flush the fragment we just finished.
        if (this.hasMdat) this.emit(out);
        this.unit.push(box);
        continue;
      }
      if (type === "mdat") {
        this.unit.push(box);
        this.hasMdat = true;
        continue;
      }
      // Other top-level boxes (sidx/free/…) ride along with the open unit.
      if (this.unit.length > 0) this.unit.push(box);
    }

    this.pending = bytes.slice(offset);
    return out;
  }

  /** Emit whatever is complete when the stream ends. */
  flush(): Bytes[] {
    const out: Bytes[] = [];
    if (this.pending.length > 0) {
      this.unit.push(this.pending);
      this.pending = new Uint8Array(0);
    }
    this.emit(out);
    return out;
  }

  private emit(out: Bytes[]): void {
    if (this.unit.length === 0) return;
    out.push(join(this.unit));
    this.unit = [];
    this.hasMdat = false;
  }
}

/* -------------------- streamer -------------------- */

export interface MseStreamerCallbacks {
  /** End of the appended buffer in engine seconds (relative to stream start). */
  onBufferedEnd: (endSeconds: number) => void;
  onError: (message: string) => void;
  /**
   * The server rejected the stream because it is already transcoding at
   * capacity (HTTP 429). The player uses this to fall back to original
   * playback instead of showing a hard error.
   */
  onBusy?: (retryAfterSeconds: number | null) => void;
}

export interface MseStreamer {
  /** Attach a fresh MediaSource and start fetching `url`. */
  start(url: string, autoplay: boolean): void;
  /** Abort fetching, detach the element and release the source. */
  dispose(): void;
}

/**
 * One MSE playback session. Each seek / quality switch tears the previous
 * session down and starts a new one, mirroring the native restart-on-seek
 * model, so the controller's virtual timeline math is unchanged.
 */
export function createMseStreamer(
  video: HTMLVideoElement,
  callbacks: MseStreamerCallbacks
): MseStreamer {
  let disposed = false;
  const isDisposed = (): boolean => disposed;
  let objectUrl: string | null = null;
  let media: MediaSource | null = null;
  let managed: ManagedMediaSourceLike | null = null;
  let buffer: SourceBuffer | null = null;
  let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
  let abort: AbortController | null = null;
  const splitter = new Fmp4Splitter();
  const queue: Bytes[] = [];
  let streamDone = false;
  let reading = false;
  let wants = true;
  let autoplayWanted = false;

  const updateBuffered = () => {
    if (!buffer || buffer.buffered.length === 0) return;
    callbacks.onBufferedEnd(buffer.buffered.end(buffer.buffered.length - 1));
  };

  const bufferedAhead = (): number => {
    if (!buffer || buffer.buffered.length === 0) return 0;
    return buffer.buffered.end(buffer.buffered.length - 1) - video.currentTime;
  };

  const shouldRead = (): boolean => {
    if (isDisposed() || streamDone || !reader) return false;
    // ManagedMediaSource asks for data through start/endstreaming.
    if (managed) return wants;
    return wants && bufferedAhead() < HIGH_WATER_SECONDS;
  };

  const maybeEnd = () => {
    if (isDisposed() || !media || !buffer) return;
    if (
      streamDone &&
      queue.length === 0 &&
      !buffer.updating &&
      media.readyState === "open"
    ) {
      try {
        media.endOfStream();
      } catch {
        // Already ended or detached.
      }
    }
  };

  const evict = () => {
    if (isDisposed() || !buffer || buffer.updating) return;
    const keepFrom = Math.max(0, video.currentTime - KEEP_BEHIND_SECONDS);
    if (buffer.buffered.length > 0 && buffer.buffered.start(0) < keepFrom) {
      try {
        buffer.remove(0, keepFrom);
      } catch {
        // Ignore races with the UA evicting on its own.
      }
      return;
    }
    // The forward-buffer cap should make this unreachable; stop cleanly.
    callbacks.onError("Video buffer is full on this device.");
    queue.length = 0;
  };

  const pumpAppend = () => {
    if (isDisposed() || !buffer || buffer.updating || queue.length === 0) {
      maybeEnd();
      return;
    }
    try {
      buffer.appendBuffer(queue[0]);
      queue.shift();
    } catch (error) {
      if (error instanceof DOMException && error.name === "QuotaExceededError") {
        evict();
        return;
      }
      queue.shift();
      callbacks.onError("Unable to play compressed stream.");
    }
  };

  const pull = async (): Promise<void> => {
    if (isDisposed() || reading || !shouldRead()) return;
    const current = reader;
    if (!current) return;
    reading = true;
    let result: ReadableStreamReadResult<Uint8Array>;
    try {
      result = await current.read();
    } catch {
      // Aborted by dispose() or the connection dropped; stop silently.
      reading = false;
      return;
    }
    reading = false;
    if (isDisposed()) return;

    if (result.done) {
      streamDone = true;
      for (const unit of splitter.flush()) queue.push(unit);
      pumpAppend();
      return;
    }
    for (const unit of splitter.push(result.value)) queue.push(unit);
    pumpAppend();
    void pull();
  };

  const onUpdateEnd = () => {
    if (isDisposed()) return;
    updateBuffered();
    if (queue.length > 0) pumpAppend();
    else maybeEnd();
    void pull();
  };

  const onBufferError = () => {
    if (!isDisposed()) callbacks.onError("Unable to play compressed stream.");
  };

  const onStartStreaming = () => {
    wants = true;
    void pull();
  };

  const onEndStreaming = () => {
    wants = false;
  };

  const onTimeUpdate = () => {
    // Plain MediaSource has no streaming signal: top up as the playhead moves.
    if (!managed) void pull();
  };

  const onSourceOpen = () => {
    if (isDisposed() || !media) return;
    const codec = pickMseCodec();
    if (!codec) {
      callbacks.onError("Compressed playback is not supported on this device.");
      return;
    }
    let created: SourceBuffer;
    try {
      created = media.addSourceBuffer(codec);
    } catch {
      callbacks.onError("Compressed playback is not supported on this device.");
      return;
    }
    buffer = created;
    created.addEventListener("updateend", onUpdateEnd);
    created.addEventListener("error", onBufferError);
    if (autoplayWanted && video.paused) {
      video.play().catch(ignore);
    }
    void pull();
  };

  const fetchStream = async (url: string): Promise<void> => {
    abort = new AbortController();
    let response: Response;
    try {
      response = await fetch(url, {
        signal: abort.signal,
        // Match the app's axios posture (withCredentials) so cross-origin
        // cookie auth keeps working behind the CORS config.
        credentials: "include",
      });
    } catch {
      if (!isDisposed()) callbacks.onError("Unable to load compressed stream.");
      return;
    }
    if (!response.ok || !response.body) {
      if (isDisposed()) return;
      if (response.status === 429 && callbacks.onBusy) {
        const raw = response.headers.get("Retry-After");
        const retryAfter = raw === null ? null : Number(raw);
        callbacks.onBusy(
          retryAfter !== null && Number.isFinite(retryAfter) ? retryAfter : null
        );
      } else {
        callbacks.onError("Unable to load compressed stream.");
      }
      return;
    }
    reader = response.body.getReader();
    void pull();
  };

  const start = (url: string, autoplay: boolean): void => {
    const resolved = resolveMseConstructor();
    if (!resolved) {
      callbacks.onError("Compressed playback is not supported on this device.");
      return;
    }
    autoplayWanted = autoplay;

    const source = new resolved.ctor();
    media = source;
    if (resolved.managed) {
      const managedSource = source as ManagedMediaSourceLike;
      managed = managedSource;
      // ManagedMediaSource only activates when remote playback is disabled.
      video.disableRemotePlayback = true;
      managedSource.addEventListener("startstreaming", onStartStreaming);
      managedSource.addEventListener("endstreaming", onEndStreaming);
    }
    source.addEventListener("sourceopen", onSourceOpen);

    objectUrl = URL.createObjectURL(source);
    video.src = objectUrl;
    video.addEventListener("timeupdate", onTimeUpdate);
    if (autoplay) video.play().catch(ignore);

    void fetchStream(url);
  };

  const dispose = (): void => {
    if (isDisposed()) return;
    disposed = true;

    abort?.abort();
    if (reader) void reader.cancel().catch(ignore);
    reader = null;
    video.removeEventListener("timeupdate", onTimeUpdate);

    if (managed) {
      managed.removeEventListener("startstreaming", onStartStreaming);
      managed.removeEventListener("endstreaming", onEndStreaming);
      managed = null;
    }
    if (media) media.removeEventListener("sourceopen", onSourceOpen);
    if (buffer) {
      buffer.removeEventListener("updateend", onUpdateEnd);
      buffer.removeEventListener("error", onBufferError);
    }
    if (media && buffer && media.readyState === "open") {
      try {
        media.removeSourceBuffer(buffer);
      } catch {
        // Already detached.
      }
    }
    buffer = null;
    media = null;

    if (objectUrl) {
      // Detach before revoking so the element cannot fire an error for a
      // dead blob while a new source is being attached.
      if (video.src === objectUrl) {
        video.removeAttribute("src");
        video.load();
      }
      URL.revokeObjectURL(objectUrl);
      objectUrl = null;
    }
    queue.length = 0;
  };

  return { start, dispose };
}
