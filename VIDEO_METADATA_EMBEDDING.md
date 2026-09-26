# How `vid_metadata` JSON is embedded into MP4 metadata

This document traces how an AI-pipeline sidecar
`.vid_metadata/<video filename>_timestamps.json` is embedded into an
MP4/MOV container's metadata tags, and how it is read back.

The pipeline is **sidecar JSON → ffmpeg `-metadata` tags → reader → API**.

## Overview

```
.vid_metadata/<file>_timestamps.json
        │  parseEventsJSON()  (tolerant normalization)
        ▼
buildEmbedPayload()  →  {"video":…, "events":[[s,e],…], "scenes":[{start,end},…]}
        │  metadata["description"] = payload
        │  metadata["comment"]     = payload
        ▼
RemuxVideoWithMetadata()  →  ffmpeg -c copy -metadata description=… -metadata comment=…
        │  (mp4/mov/m4v only, +faststart, atomic temp + rename)
        ▼
moov/udta/meta/ilst  atoms:  desc / ©cmt
        │  ReadMP4Tags()  (hand-rolled box parser)
        ▼
readEmbeddedVideoEvents()  →  GET /api/user/video/metadata/file/{filepath}
```

Accessors:

- Write path: `backend/internal/util/video_remux.go`,
  `backend/internal/service/video_rename_done.go`
- Read path: `backend/internal/util/mp4_tags.go`,
  `backend/internal/service/video_metadata.go`

---

## 1. Decide whether to embed — `backend/internal/service/video_rename_done.go`

`processVideoRenameDone` reads the sidecar, builds a portable JSON payload, and
only embeds it if the file is an MP4-family container:

```go
// video_rename_done.go:176-179
_, sidecarStatErr := os.Stat(sidecarPath)
sidecarExists := sidecarStatErr == nil
payload, hasMetadata := buildEmbedPayload(sidecarPath, newName)
willEmbed := hasMetadata && util.IsMP4FamilyExt(filepath.Ext(procPath))
```

The payload is a **superset** of the original sidecar: the `events` pairs the
player parses plus a `scenes` array for desktop/doc consumers and the final
filename. `parseEventsJSON` is reused here, so any accepted input shape is
normalized first:

```go
// video_rename_done.go:237-262
func buildEmbedPayload(sidecarPath, newName string) (string, bool) {
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return "", false
	}

	events, ok := parseEventsJSON(data)
	if !ok {
		return "", false
	}

	scenes := make([]map[string]float64, 0, len(events))
	for _, e := range events {
		scenes = append(scenes, map[string]float64{"start": e[0], "end": e[1]})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"video":  newName,
		"events": events,
		"scenes": scenes,
	})
	if err != nil {
		return "", false
	}
	return string(payload), true
}
```

The payload is stuffed into **two** MP4 tags (`description` and `comment`), then
handed to the remuxer together with any rotation delta:

```go
// video_rename_done.go:195-213
if rotateAngle != 0 || willEmbed {
	metadata := map[string]string{}
	if willEmbed {
		metadata["description"] = payload
		metadata["comment"] = payload
	}
	if err := util.RemuxVideoWithMetadata(context.Background(), procPath, destPath, rotateAngle, metadata, tracker.Update); err != nil {
		return failProcessing(err)
	}
	// Remux succeeded: drop the staging file, the annotated copy is in done/.
	if err := os.Remove(procPath); err != nil {
		logger.L.Warn("failed to remove staging file after remux", "path", procPath, "err", err)
	}
} else {
	// Nothing to process (no rotation and nothing to embed): just move it.
	if err := os.Rename(procPath, destPath); err != nil {
		return failProcessing(fmt.Errorf("failed to move file to done: %w", err))
	}
}
```

On success the sidecar is deleted (the container tag is now the source of
truth); if embedding was skipped the sidecar is moved into
`done/.vid_metadata/` instead, where its presence is the "not embedded" signal
(`video_rename_done.go:220-228`).

---

## 2. The actual ffmpeg invocation — `backend/internal/util/video_remux.go`

MP4 family is defined here, and the tags are written with plain
`-metadata key=value`:

```go
// video_remux.go:72-84
func IsMP4FamilyExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp4", ".mov", ".m4v":
		return true
	default:
		return false
	}
}
```

The core `ffmpeg` argument construction (keys sorted for deterministic output):

```go
// video_remux.go:118-146
args := []string{"--as=524288000", "ffmpeg", "-y", "-loglevel", "error"}

newAngle := 0
if rotateAngle != 0 {
	current, err := ProbeVideoRotation(ctx, srcPath)
	if err != nil {
		return err
	}
	newAngle = AdjustRotationAngle(current, rotateAngle)
	args = append(args, "-display_rotation", fmt.Sprintf("%d", newAngle))
}

args = append(args, "-i", srcPath, "-map", "0", "-c", "copy")

if rotateAngle != 0 {
	args = append(args, "-metadata:s:v:0", fmt.Sprintf("rotate=%d", newAngle))
}

keys := make([]string, 0, len(metadata))
for k := range metadata {
	keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
	args = append(args, "-metadata", fmt.Sprintf("%s=%s", k, metadata[k]))
}

args = append(args, "-movflags", "+faststart", tmpPath)
```

So the command that actually embeds the JSON is effectively:

```bash
prlimit --as=524288000 ffmpeg -y -loglevel error \
  -i .cloud_reserve/temp/<name>.mp4 -map 0 -c copy \
  -metadata description='{"video":"new.mp4","events":[[s,e],...],"scenes":[{"start":s,"end":e},...]}' \
  -metadata comment='{"video":"new.mp4","events":[[s,e],...],"scenes":[{"start":s,"end":e},...]}' \
  -movflags +faststart .embed-<pid>-<nanos>.tmp.mp4
```

Key points in that block:

- **`-c copy`** — no re-encode; only the container/tag layer is rewritten.
- **`-metadata description=…` / `-metadata comment=…`** — ffmpeg's MP4/MOV
  muxer maps these to the QuickTime `desc` / `©cmt` atoms.
- **`-movflags +faststart`** — moves the `moov` atom to the front, which is also
  where those tags live.
- The output is a hidden temp file in the destination directory (same
  filesystem), then atomically `os.Rename`d over `destPath` on success
  (`video_remux.go:191-193`); `srcPath` is untouched on failure.

Wrapping and cleanup around the process:

```go
// video_remux.go:110-116
ext := filepath.Ext(destPath)
tmpPath := filepath.Join(destDir, fmt.Sprintf(".embed-%d-%d.tmp%s", os.Getpid(), time.Now().UnixNano(), ext))
defer func() {
	if _, err := os.Stat(tmpPath); err == nil {
		_ = os.Remove(tmpPath)
	}
}()
```

```go
// video_remux.go:148-155
cmd := NewCommand(ctx, "prlimit", args...)
var stderr bytes.Buffer
cmd.Stderr = &stderr
...
if err := cmd.Start(); err != nil {
	return fmt.Errorf("failed to start ffmpeg: %w", err)
}
```

`NewCommand` (`backend/internal/util/exec.go`) wraps everything in `prlimit` for
the 500 MB address-space cap and manages the process group so a cancelled remux
does not orphan ffmpeg children. Progress is emitted by polling the growing
temp file every 250 ms (`video_remux.go:160-185`).

---

## 3. Read-back — `backend/internal/util/mp4_tags.go` + `backend/internal/service/video_metadata.go`

The tags are read by a hand-rolled MP4 box parser (no ffprobe), which maps the
QuickTime atoms back to the keys used above:

```go
// mp4_tags.go:12-19
var mp4TagNames = map[string]string{
	"\xa9cmt": "comment",          // ©cmt
	"desc":    "description",      // desc
	"\xa9nam": "title",            // ©nam
	"\xa9day": "date",             // ©day
	"\xa9too": "encoder",          // ©too
	"ldes":    "long_description", // ldes
}
```

`ReadMP4Tags` walks only `moov/udta/meta/ilst` and skips `mdat` by seeking, so
it costs ~tens of KB regardless of file size. The API handler then tries
embedded tags first and falls back to the sidecar:

```go
// video_metadata.go:64-83
func readEmbeddedVideoEvents(fullPath string) ([][]float64, bool) {
	tags, err := util.ReadMP4Tags(fullPath)
	if err != nil {
		logger.L.Debug("mp4 tag read failed", "path", fullPath, "err", err)
		return nil, false
	}

	for _, key := range []string{"description", "comment"} {
		payload, ok := tags[key]
		if !ok || payload == "" {
			continue
		}
		if events, ok := parseEventsJSON([]byte(payload)); ok {
			return events, true
		}
	}
	return nil, false
}
```

`GetVideoMetadata` (`video_metadata.go:37-62`) resolves in order **embedded MP4
tag → sibling sidecar JSON → 404**, reporting `source: "embedded"` or
`source: "sidecar"` in the response.

---

## Summary of the round trip

1. `sidecar .vid_metadata/*_timestamps.json`
2. `parseEventsJSON` normalizes it to `[start, end]` pairs.
3. `buildEmbedPayload` builds `{video, events, scenes}` JSON.
4. `RemuxVideoWithMetadata` runs
   `ffmpeg -c copy -metadata description=… -metadata comment=… -movflags +faststart`.
5. Tags land in `moov/udta/meta/ilst` as `desc` / `©cmt`.
6. `ReadMP4Tags` parses them back.
7. `readEmbeddedVideoEvents` serves them to the player.

Supporting tests:

- `backend/internal/util/video_remux_test.go` — round-trips
  `description`/`comment` through real ffmpeg.
- `backend/internal/util/mp4_tags_test.go` — UTF-8/quoting and tail-`moov`
  handling.
- `backend/internal/service/video_metadata_test.go` — payload shape
  normalization.
