import { useCallback, useEffect } from "react";
import { stepPlaybackRate } from "@/utils/videoSpeed";

interface UseVideoKeyboardParams {
  /** When true the rename dialog is open and transport keys are ignored. */
  showRenameModal: boolean;
  onTogglePlay: () => void;
  onSkip: (seconds: number) => void;
  onNextEvent: () => void;
  onPrevEvent: () => void;
  onRenameSave: () => void;
  /**
   * Close the player (or cancel the rename dialog when it is open). Escape
   * calls this instead of walking browser history, so the modal's own
   * `useDialogHistory` entry is popped exactly once.
   */
  onEscape: () => void;
  /**
   * Current playback rate. When `onChangeSpeed` is also provided, `<` / `,`
   * step the speed down and `>` / `.` step it up.
   */
  playbackRate?: number;
  /** Change the playback rate; enables the `<` / `>` speed shortcuts. */
  onChangeSpeed?: (rate: number) => void;
  /**
   * Highlight-editor shortcuts, present only while the editor is open:
   * `[` sets the selected event's start to the playhead, `]` its end, and
   * Delete/Backspace removes it.
   */
  eventEditor?: {
    onSetIn: () => void;
    onSetOut: () => void;
    onDelete: () => void;
  };
}

/**
 * Document-level keyboard shortcuts:
 * - ArrowLeft/Right skip ∓1s; +Shift skips ∓3s
 * - Ctrl+ArrowLeft/Right jump to the previous/next detected event
 * - Space toggles play/pause
 * - `<` / `,` lower and `>` / `.` raise the playback speed (when a speed handler is given)
 * - Enter saves the rename dialog, Escape closes the player
 */
export function useVideoKeyboard({
  showRenameModal,
  onTogglePlay,
  onSkip,
  onNextEvent,
  onPrevEvent,
  onRenameSave,
  onEscape,
  playbackRate,
  onChangeSpeed,
  eventEditor,
}: UseVideoKeyboardParams) {
  const handleKeyPress = useCallback(
    (event: KeyboardEvent) => {
      // The key property returns the character pressed
      const key: string = event.key;
      // shiftKey property is a boolean indicating if Shift was held
      const isShift: boolean = event.shiftKey;
      // Ctrl (or Cmd on macOS) is the event-navigation modifier.
      const isCtrl: boolean = event.ctrlKey || event.metaKey;

      let actionDescription = "";

      switch (key) {
        case "ArrowLeft":
          if (showRenameModal) {
            return; //dont do anything in rename
          }
          if (isCtrl) {
            actionDescription = "prevEvent";
            onPrevEvent();
          } else if (isShift) {
            actionDescription = "skip(-3)";
            onSkip(-3);
          } else {
            actionDescription = "skip(-1)";
            onSkip(-1);
          }
          break;

        case "ArrowRight":
          if (showRenameModal) {
            return; //dont do anything in rename
          }
          if (isCtrl) {
            actionDescription = "nextEvent";
            onNextEvent();
          } else if (isShift) {
            actionDescription = "skip(3)";
            onSkip(3);
          } else {
            actionDescription = "skip(1)";
            onSkip(1);
          }
          break;

        case " ": // Spacebar
          if (showRenameModal) {
            return; //dont do anything in rename
          }
          actionDescription = "togglePlay";
          onTogglePlay();
          break;

        // `<` / `,` → slower; the unshifted comma means no Shift is required.
        case "<":
        case ",":
          if (showRenameModal) {
            return;
          }
          if (!onChangeSpeed) {
            return;
          }
          actionDescription = "speedDown";
          onChangeSpeed(stepPlaybackRate(playbackRate ?? 1, -1));
          break;

        // `>` / `.` → faster; the unshifted period means no Shift is required.
        case ">":
        case ".":
          if (showRenameModal) {
            return;
          }
          if (!onChangeSpeed) {
            return;
          }
          actionDescription = "speedUp";
          onChangeSpeed(stepPlaybackRate(playbackRate ?? 1, 1));
          break;

        // `[` / `]` set the selected event's start / end to the playhead.
        case "[":
          if (showRenameModal || !eventEditor) return;
          actionDescription = "setEventStart";
          eventEditor.onSetIn();
          break;

        case "]":
          if (showRenameModal || !eventEditor) return;
          actionDescription = "setEventEnd";
          eventEditor.onSetOut();
          break;

        // Remove the selected event. Backspace is included because it is the
        // muscle-memory "delete" on Mac, and preventDefault stops its browser
        // history-back behaviour.
        case "Delete":
        case "Backspace":
          if (showRenameModal || !eventEditor) return;
          actionDescription = "deleteEvent";
          eventEditor.onDelete();
          event.preventDefault();
          break;

        case "Enter":
          if (showRenameModal) {
            actionDescription = "handleRenameSave";
            onRenameSave();
          }
          break;

        case "Escape":
          actionDescription = "handleDismiss";
          onEscape();
          break;
        default:
          // Ignore other keys
          return;
      }

      // Prevent default browser actions for navigation/scrolling keys when not editing name
      if (
        !showRenameModal &&
        (key === "ArrowLeft" || key === "ArrowRight" || key === " ")
      ) {
        event.preventDefault();
      }
      console.log(`Key Pressed: ${actionDescription}`);
    },
    [
      showRenameModal,
      onTogglePlay,
      onSkip,
      onNextEvent,
      onPrevEvent,
      onRenameSave,
      onEscape,
      playbackRate,
      onChangeSpeed,
      eventEditor,
    ]
  );

  useEffect(() => {
    // Add the keydown listener to the document
    document.addEventListener("keydown", handleKeyPress as (e: Event) => void);
    // Cleanup: Remove the event listener when the component unmounts
    return () => {
      document.removeEventListener(
        "keydown",
        handleKeyPress as (e: Event) => void
      );
    };
  }, [handleKeyPress]);
}
