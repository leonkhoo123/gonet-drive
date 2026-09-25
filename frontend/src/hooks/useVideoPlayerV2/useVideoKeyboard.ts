import { useCallback, useEffect } from "react";

interface UseVideoKeyboardParams {
  /** When true the rename dialog is open and transport keys are ignored. */
  showRenameModal: boolean;
  onTogglePlay: () => void;
  onSkip: (seconds: number) => void;
  onNextEvent: () => void;
  onPrevEvent: () => void;
  onRenameSave: () => void;
}

/**
 * Document-level keyboard shortcuts:
 * - ArrowLeft/Right (+Shift) skip ∓1/∓3s
 * - Space toggles play/pause
 * - ,/< and ./> or p/P and n/N jump between detected events
 * - Enter saves the rename dialog, Escape triggers back-navigation
 */
export function useVideoKeyboard({
  showRenameModal,
  onTogglePlay,
  onSkip,
  onNextEvent,
  onPrevEvent,
  onRenameSave,
}: UseVideoKeyboardParams) {
  const handleKeyPress = useCallback(
    (event: KeyboardEvent) => {
      // The key property returns the character pressed
      const key: string = event.key;
      // shiftKey property is a boolean indicating if Shift was held
      const isShift: boolean = event.shiftKey;

      let actionDescription = "";

      switch (key) {
        case "ArrowLeft":
          if (showRenameModal) {
            return; //dont do anything in rename
          }
          if (isShift) {
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
          if (isShift) {
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

        case "<":
        case ",": // Shift+comma / comma → previous event start
          if (showRenameModal) {
            return;
          }
          actionDescription = "prevEvent";
          onPrevEvent();
          break;

        case ">":
        case ".": // Shift+period / period → next event start
          if (showRenameModal) {
            return;
          }
          actionDescription = "nextEvent";
          onNextEvent();
          break;

        case "n":
        case "N":
          if (showRenameModal) {
            return;
          }
          actionDescription = "nextEvent";
          onNextEvent();
          break;

        case "p":
        case "P":
          if (showRenameModal) {
            return;
          }
          actionDescription = "prevEvent";
          onPrevEvent();
          break;

        case "Enter":
          if (showRenameModal) {
            actionDescription = "handleRenameSave";
            onRenameSave();
          }
          break;

        case "Escape":
          actionDescription = "handleDismiss";
          // handleDismiss();
          window.history.back();
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
