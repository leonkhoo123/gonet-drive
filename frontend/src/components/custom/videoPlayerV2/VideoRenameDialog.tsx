import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CONTROL, CONTROL_GREEN, PANEL } from "./controlStyles";
import type { VideoRenameDialogProps } from "./types";

/**
 * Overlay dialog for renaming the current video.
 *
 * The name is built from dash-joined segments: type a word, press Enter and it
 * lands in the preview as a chip. So "nice" + Enter, "pink" + Enter, "sky" +
 * Enter composes "nice-pink-sky", which Done commits (the hook appends the
 * extension). Chips can be deleted (×), tapped to pull back into the input for
 * editing, or dragged to reorder.
 */
export function VideoRenameDialog({
  tempName,
  onChangeName,
  onDefault,
  onCancel,
  onSave,
}: VideoRenameDialogProps) {
  const [draft, setDraft] = useState("");
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  const segments = tempName ? tempName.split("-").filter(Boolean) : [];

  const commitSegments = (next: string[]) => {
    onChangeName(next.join("-"));
  };

  /** Append the typed word to the composed name and clear the input. */
  const appendSegment = () => {
    const segment = draft.trim();
    if (!segment) return;
    commitSegments([...segments, segment]);
    setDraft("");
  };

  /** Drop a chip from the composed name. */
  const removeSegment = (index: number) => {
    commitSegments(segments.filter((_, i) => i !== index));
  };

  /** Pull a chip back into the input so it can be retyped or repositioned. */
  const editSegment = (index: number) => {
    setDraft(segments[index]);
    removeSegment(index);
  };

  /** Reorder a chip; dragging the last word to the end keeps it in place. */
  const moveSegment = (from: number, to: number) => {
    if (from === to) return;
    const next = [...segments];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    commitSegments(next);
  };

  /** Commit, including any word still sitting unsubmitted in the input. */
  const handleDone = () => {
    const segment = draft.trim();
    onSave(segment ? [...segments, segment].join("-") : tempName);
  };

  return (
    <div className="absolute inset-0 z-40 bg-black/70 flex items-center justify-center">
      <div className={`rounded-md p-4 w-full max-w-5xl mx-2 ${PANEL}`}>
        <h3 className="font-semibold mb-2">Name</h3>

        {/* Composed name preview. Each segment is a chip; the empty state shows
            a placeholder until the first word is added. */}
        <div className="mb-2 min-h-10 w-full p-2 text-base flex flex-wrap items-center gap-1.5">
          {segments.length === 0 ? (
            <span className="text-white/40">Full name</span>
          ) : (
            segments.map((segment, index) => {
              // Duplicate words are legal, so suffix repeats with a counter
              // instead of using the array index as the key.
              const occurrence = segments
                .slice(0, index)
                .filter((s) => s === segment).length;
              const key =
                occurrence === 0 ? segment : `${segment}#${String(occurrence)}`;
              return (
                <span
                  key={key}
                  draggable
                  onDragStart={() => {
                    setDragIndex(index);
                  }}
                  onDragOver={(e) => {
                    e.preventDefault();
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    if (dragIndex !== null) moveSegment(dragIndex, index);
                    setDragIndex(null);
                  }}
                  onDragEnd={() => {
                    setDragIndex(null);
                  }}
                  className={`inline-flex items-center gap-1 rounded bg-white/10 py-0.5 pl-2 pr-1 ${
                    dragIndex === index ? "opacity-50" : ""
                  }`}
                >
                  <button
                    type="button"
                    onClick={() => {
                      editSegment(index);
                    }}
                    className="max-w-[16rem] truncate text-left hover:text-white/70"
                    title="Tap to edit"
                  >
                    {segment}
                  </button>
                  <button
                    type="button"
                    aria-label={`Remove ${segment}`}
                    onClick={() => {
                      removeSegment(index);
                    }}
                    className="rounded p-0.5 text-white/60 hover:bg-white/15 hover:text-white"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </span>
              );
            })
          )}
        </div>

        {/* The form is what makes the iPadOS/Android on-screen Return key work:
            both submit the single-input form even when their keydown reports an
            "Unidentified" key (common on Android IMEs). The keydown handler
            covers desktop and cancels the implicit submit so it never runs
            twice. */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            appendSegment();
          }}
        >
          <input
            type="text"
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                // Enter adds a segment rather than submitting the dialog.
                e.preventDefault();
                e.stopPropagation();
                appendSegment();
              }
            }}
            className="w-full rounded-md border border-white/15 bg-black/50 p-2 mb-4 text-base text-white placeholder:text-white/40 outline-none focus:border-white/40 focus:ring-2 focus:ring-white/20"
            placeholder="Type a word and press Enter"
            autoFocus
            enterKeyHint="done"
          />
        </form>

        <div className="flex justify-between">
          <Button variant="ghost" onClick={onDefault} className={CONTROL}>
            Default
          </Button>
          <div className="space-x-2">
            <Button variant="ghost" onClick={onCancel} className={CONTROL}>
              Cancel
            </Button>
            <Button variant="ghost" onClick={handleDone} className={CONTROL_GREEN}>
              Done
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
