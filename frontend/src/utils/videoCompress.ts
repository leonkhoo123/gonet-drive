/** Path fragment of the normal (non-compressed) video play endpoint. */
const PLAY_FRAGMENT = "/video/play/file";
/** Path fragment of the on-the-fly compression stream endpoint. */
const STREAM_FRAGMENT = "/video/stream/file";

/**
 * Rewrite a regular video URL into the on-the-fly compression stream URL at an
 * absolute start offset and target resolution (the shorter side in px). The
 * player swaps this URL on every seek because each stream is a fresh transcode
 * that always begins at 0.
 */
export function buildCompressStreamUrl(
  fileUrl: string,
  startSeconds: number,
  shortSide: number
): string {
  const base = fileUrl.replace(PLAY_FRAGMENT, STREAM_FRAGMENT);
  const separator = base.includes("?") ? "&" : "?";
  const start = Math.max(0, Math.floor(startSeconds));
  return `${base}${separator}start=${String(start)}&quality=${String(shortSide)}`;
}
