import { useEffect, useRef, useId } from 'react';

/**
 * Hook to manage history state for dialogs/modals.
 * Pushes a state when opened so the Android Back button closes the dialog
 * instead of navigating to the previous page.
 */
export function useDialogHistory(isOpen: boolean, onClose: () => void) {
  const dialogId = useId();
  const isBackButtonClicked = useRef(false);
  const onCloseRef = useRef(onClose);

  // Keep ref updated to avoid stale closures
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const uniqueInstanceId = Math.random().toString(36).substring(2, 9);
    const dialogStateKey = `dialog_${dialogId}`;

    // Push a new state when the dialog opens
    // We preserve the existing history state to avoid breaking routers
    const currentState = (window.history.state as Record<string, unknown> | null) ?? {};
    const stateToPush = { ...currentState, [dialogStateKey]: uniqueInstanceId };
    window.history.pushState(stateToPush, '');

    let isMounted = true;

    const handlePopState = (e: PopStateEvent) => {
      // When the user presses the back button, the browser pops our pushed state.
      // React Router might wrap state in e.state.usr
      const state = e.state as Record<string, unknown> | null;
      const actualState = (state?.usr !== undefined ? state.usr : state) as Record<string, unknown> | null;

      if (actualState?.[dialogStateKey] !== uniqueInstanceId) {
        isBackButtonClicked.current = true;
        onCloseRef.current();
      }
    };

    // Delay adding the listener to avoid catching immediate popstate events
    // triggered by React 18 Strict Mode's unmount `history.back()`
    const timerId = setTimeout(() => {
      if (isMounted) {
        window.addEventListener('popstate', handlePopState);
      }
    }, 100);

    return () => {
      isMounted = false;
      clearTimeout(timerId);
      window.removeEventListener('popstate', handlePopState);

      // If the dialog was closed by a UI action (e.g. clicking a "Close" button),
      // our pushed state is still at the top of the history stack.
      // We need to pop it programmatically.
      if (!isBackButtonClicked.current) {
        const hState = window.history.state as Record<string, unknown> | null;
        const actualHState = (hState?.usr !== undefined ? hState.usr : hState) as Record<string, unknown> | null;
        
        if (actualHState?.[dialogStateKey] === uniqueInstanceId) {
          window.history.back();
        }
      }
      isBackButtonClicked.current = false;
    };
  }, [isOpen, dialogId]);
}
