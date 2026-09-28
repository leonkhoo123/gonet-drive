import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent, ReactNode } from "react";
import { cn } from "@/lib/utils";
import {
  findAllMatches,
  getByteSize,
  HIGHLIGHT_MAX_BYTES,
  HIGHLIGHT_MAX_MATCHES,
} from "./helpers";

interface UseFindInFileOptions {
  /** Current editor contents (the draft). */
  draft: string;
  /** Whether the modal is in edit mode; matching only runs while editing. */
  isEditing: boolean;
}

/**
 * Find-in-file state and behavior for the text editor: match computation,
 * highlight mirror sizing/scrolling, and keyboard navigation.
 */
export function useFindInFile({ draft, isEditing }: UseFindInFileOptions) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [matchCase, setMatchCase] = useState(false);
  const [activeMatch, setActiveMatch] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const highlightRef = useRef<HTMLDivElement>(null);
  const activeMarkRef = useRef<HTMLElement>(null);
  const [mirrorSize, setMirrorSize] = useState<{ width: number; height: number } | null>(null);

  const draftBytes = useMemo(() => (isEditing ? getByteSize(draft) : 0), [isEditing, draft]);

  const matches = useMemo(
    () => (isEditing && searchOpen ? findAllMatches(draft, searchQuery, matchCase) : []),
    [isEditing, searchOpen, draft, searchQuery, matchCase],
  );
  const highlightActive = matches.length === 0 || activeMatch < 0
    ? -1
    : Math.min(activeMatch, matches.length - 1);
  const mirrorEnabled = searchOpen && searchQuery !== "" && matches.length > 0 &&
    matches.length <= HIGHLIGHT_MAX_MATCHES && draftBytes <= HIGHLIGHT_MAX_BYTES;

  const highlightedContent = useMemo(() => {
    if (!mirrorEnabled) return null;
    const queryLength = searchQuery.length;
    const nodes: ReactNode[] = [];
    let cursor = 0;
    matches.forEach((start, i) => {
      if (start > cursor) nodes.push(draft.slice(cursor, start));
      const isActive = i === highlightActive;
      nodes.push(
        <mark
          key={start}
          ref={isActive ? activeMarkRef : undefined}
          className={cn(
            "rounded-sm text-transparent",
            isActive ? "bg-orange-400/60" : "bg-yellow-300/40",
          )}
        >
          {draft.slice(start, start + queryLength)}
        </mark>,
      );
      cursor = start + queryLength;
    });
    if (cursor < draft.length) nodes.push(draft.slice(cursor));
    return nodes;
  }, [mirrorEnabled, matches, draft, searchQuery, highlightActive]);

  const syncMirror = useCallback(() => {
    const textarea = textareaRef.current;
    const mirror = highlightRef.current;
    if (!textarea) return;
    const next = { width: textarea.clientWidth, height: textarea.clientHeight };
    setMirrorSize((prev) =>
      prev && prev.width === next.width && prev.height === next.height ? prev : next,
    );
    if (mirror) {
      mirror.scrollTop = textarea.scrollTop;
      mirror.scrollLeft = textarea.scrollLeft;
    }
  }, []);

  // Keep the highlight mirror aligned with the textarea's content box and scroll.
  useLayoutEffect(() => {
    if (!mirrorEnabled) return;
    syncMirror();
  }, [mirrorEnabled, draft, searchQuery, syncMirror]);

  // Re-align when the editor itself is resized (window resize, fullscreen, ...).
  useEffect(() => {
    if (!mirrorEnabled) return;
    const textarea = textareaRef.current;
    if (!textarea) return;
    const observer = new ResizeObserver(() => { syncMirror(); });
    observer.observe(textarea);
    window.addEventListener('resize', syncMirror);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', syncMirror);
    };
  }, [mirrorEnabled, syncMirror]);

  // Scroll the editor to the active match when navigating (Enter / arrows).
  useLayoutEffect(() => {
    if (!mirrorEnabled) return;
    const textarea = textareaRef.current;
    const mark = activeMarkRef.current;
    if (!textarea || !mark) return;
    const top = mark.offsetTop;
    const bottom = top + mark.offsetHeight;
    if (top < textarea.scrollTop || bottom > textarea.scrollTop + textarea.clientHeight) {
      textarea.scrollTop = Math.max(0, top - textarea.clientHeight / 2);
    }
    const mirror = highlightRef.current;
    if (mirror) {
      mirror.scrollTop = textarea.scrollTop;
      mirror.scrollLeft = textarea.scrollLeft;
    }
  }, [mirrorEnabled, highlightActive]);

  // Focus the find input when the bar opens.
  useEffect(() => {
    if (isEditing && searchOpen) {
      searchInputRef.current?.focus();
    }
  }, [isEditing, searchOpen]);

  const closeSearch = useCallback(() => {
    setSearchOpen(false);
    setSearchQuery("");
    setActiveMatch(0);
    textareaRef.current?.focus();
  }, []);

  const resetSearch = useCallback(() => {
    setSearchOpen(false);
    setSearchQuery("");
    setMatchCase(false);
    setActiveMatch(0);
  }, []);

  const goToMatch = (index: number) => {
    if (matches.length === 0) return;
    const next = ((index % matches.length) + matches.length) % matches.length;
    setActiveMatch(next);
    // When the mirror is active, the highlight moves on its own and a layout
    // effect scrolls to it, so focus can stay in the search box. Otherwise fall
    // back to selecting the match directly.
    if (!mirrorEnabled) {
      const textarea = textareaRef.current;
      if (textarea) {
        const start = matches[next];
        textarea.focus();
        textarea.setSelectionRange(start, start + searchQuery.length);
      }
    }
  };

  const findNext = () => { goToMatch(activeMatch < 0 ? 0 : activeMatch + 1); };
  const findPrev = () => { goToMatch(activeMatch < 0 ? matches.length - 1 : activeMatch - 1); };

  const handleSearchKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (e.shiftKey) {
        findPrev();
      } else {
        findNext();
      }
    }
  };

  const toggleSearch = () => {
    if (searchOpen) {
      closeSearch();
    } else {
      setSearchOpen(true);
    }
  };

  const syncHighlightScroll = () => {
    const textarea = textareaRef.current;
    const mirror = highlightRef.current;
    if (textarea && mirror) {
      mirror.scrollTop = textarea.scrollTop;
      mirror.scrollLeft = textarea.scrollLeft;
    }
  };

  return {
    searchOpen,
    setSearchOpen,
    searchQuery,
    setSearchQuery,
    matchCase,
    setMatchCase,
    activeMatch,
    setActiveMatch,
    textareaRef,
    searchInputRef,
    highlightRef,
    mirrorSize,
    draftBytes,
    matches,
    highlightActive,
    mirrorEnabled,
    highlightedContent,
    closeSearch,
    resetSearch,
    findNext,
    findPrev,
    handleSearchKeyDown,
    toggleSearch,
    syncHighlightScroll,
  };
}
