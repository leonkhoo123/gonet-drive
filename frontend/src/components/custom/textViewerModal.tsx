import { useState, useEffect, useCallback, useMemo, useRef, useLayoutEffect } from "react";
import { X, Download, Copy, Check, ChevronDown, ChevronUp, FileText, Pencil, Save, Loader2, AlertTriangle, Search } from "lucide-react";
import { type FileInterface, downloadFiles, saveTextFile, MAX_TEXT_EDIT_SIZE } from "@/api/api-file";
import { formatBytes } from "@/utils/utils";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import axiosLayer from "@/api/axiosLayer";
import { Light as SyntaxHighlighter } from "react-syntax-highlighter";
import { vs2015, vs } from "react-syntax-highlighter/dist/esm/styles/hljs";
import { useTheme } from "@/components/theme-provider";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// Import common languages to keep bundle size reasonable
import json from "react-syntax-highlighter/dist/esm/languages/hljs/json";
import markdown from "react-syntax-highlighter/dist/esm/languages/hljs/markdown";
import javascript from "react-syntax-highlighter/dist/esm/languages/hljs/javascript";
import typescript from "react-syntax-highlighter/dist/esm/languages/hljs/typescript";
import go from "react-syntax-highlighter/dist/esm/languages/hljs/go";
import bash from "react-syntax-highlighter/dist/esm/languages/hljs/bash";
import yaml from "react-syntax-highlighter/dist/esm/languages/hljs/yaml";
import xml from "react-syntax-highlighter/dist/esm/languages/hljs/xml";
import css from "react-syntax-highlighter/dist/esm/languages/hljs/css";
import python from "react-syntax-highlighter/dist/esm/languages/hljs/python";
import sql from "react-syntax-highlighter/dist/esm/languages/hljs/sql";
import java from "react-syntax-highlighter/dist/esm/languages/hljs/java";
import cpp from "react-syntax-highlighter/dist/esm/languages/hljs/cpp";
import dockerfile from "react-syntax-highlighter/dist/esm/languages/hljs/dockerfile";

SyntaxHighlighter.registerLanguage("json", json);
SyntaxHighlighter.registerLanguage("markdown", markdown);
SyntaxHighlighter.registerLanguage("javascript", javascript);
SyntaxHighlighter.registerLanguage("typescript", typescript);
SyntaxHighlighter.registerLanguage("go", go);
SyntaxHighlighter.registerLanguage("bash", bash);
SyntaxHighlighter.registerLanguage("yaml", yaml);
SyntaxHighlighter.registerLanguage("xml", xml);
SyntaxHighlighter.registerLanguage("css", css);
SyntaxHighlighter.registerLanguage("python", python);
SyntaxHighlighter.registerLanguage("sql", sql);
SyntaxHighlighter.registerLanguage("java", java);
SyntaxHighlighter.registerLanguage("cpp", cpp);
SyntaxHighlighter.registerLanguage("dockerfile", dockerfile);

interface TextViewerModalProps {
  file: FileInterface | null;
  isOpen: boolean;
  onClose: () => void;
  /** Enables the edit/save flow. Personal files only; share viewers leave this off. */
  editable?: boolean;
  /** Called after a successful save so the parent can refresh its listing. */
  onSaved?: () => void;
}

interface Baseline {
  size: number;
  modified: string;
}

interface ConfirmState {
  title: string;
  description: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
}

const SUPPORTED_LANGUAGES = [
  { id: "text", name: "Plain Text" },
  { id: "json", name: "JSON" },
  { id: "markdown", name: "Markdown" },
  { id: "javascript", name: "JavaScript" },
  { id: "typescript", name: "TypeScript" },
  { id: "go", name: "Go" },
  { id: "bash", name: "Bash/Shell" },
  { id: "yaml", name: "YAML" },
  { id: "xml", name: "XML/HTML" },
  { id: "css", name: "CSS" },
  { id: "python", name: "Python" },
  { id: "sql", name: "SQL" },
  { id: "java", name: "Java" },
  { id: "cpp", name: "C/C++" },
  { id: "dockerfile", name: "Dockerfile" },
];

function getLanguageFromExtension(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase();
  switch (ext) {
    case 'json': return 'json';
    case 'md':
    case 'markdown': return 'markdown';
    case 'js':
    case 'jsx': return 'javascript';
    case 'ts':
    case 'tsx': return 'typescript';
    case 'go': return 'go';
    case 'sh':
    case 'bash':
    case 'zsh': return 'bash';
    case 'yaml':
    case 'yml': return 'yaml';
    case 'xml':
    case 'html':
    case 'htm': return 'xml';
    case 'css': return 'css';
    case 'py': return 'python';
    case 'sql': return 'sql';
    case 'java': return 'java';
    case 'c':
    case 'cpp':
    case 'h':
    case 'hpp': return 'cpp';
    case 'dockerfile': return 'dockerfile';
    default:
      if (filename.toLowerCase() === 'dockerfile') return 'dockerfile';
      if (filename.toLowerCase() === 'makefile') return 'bash';
      return 'text';
  }
}

/** Accurate UTF-8 byte length (String.length counts UTF-16 code units). */
function getByteSize(value: string): number {
  return new Blob([value]).size;
}

/** All non-overlapping start indices of `query` within `text`. */
function findAllMatches(text: string, query: string, matchCase: boolean): number[] {
  if (!query) return [];
  const haystack = matchCase ? text : text.toLowerCase();
  const needle = matchCase ? query : query.toLowerCase();
  const result: number[] = [];
  let idx = haystack.indexOf(needle);
  while (idx !== -1) {
    result.push(idx);
    idx = haystack.indexOf(needle, idx + needle.length);
  }
  return result;
}

// While searching, matches are highlighted by a mirrored layer rendered behind
// the (transparent-background) textarea, so the search input keeps focus and
// typing/shortcuts stay usable. Highlighting is skipped for very large
// documents or match counts to keep typing responsive; jumping then falls back
// to selecting the match directly in the textarea.
const HIGHLIGHT_MAX_MATCHES = 2000;
const HIGHLIGHT_MAX_BYTES = 512 * 1024;

function getErrorStatus(err: unknown): number | undefined {
  if (err && typeof err === "object" && "response" in err) {
    return (err as { response?: { status?: number } }).response?.status;
  }
  return undefined;
}

function getErrorMessage(err: unknown): string {
  if (err && typeof err === "object" && "response" in err) {
    const data = (err as { response?: { data?: { error?: string } } }).response?.data;
    if (data && typeof data.error === "string" && data.error) return data.error;
  }
  return "Failed to save file. Please try again.";
}

/** Extract the on-disk state the backend included with a 409 conflict. */
function getConflictBaseline(err: unknown): Baseline | null {
  if (err && typeof err === "object" && "response" in err) {
    const data = (err as { response?: { data?: { data?: { size?: unknown; modified?: unknown } } } })
      .response?.data?.data;
    if (data && typeof data.size === "number" && typeof data.modified === "string") {
      return { size: data.size, modified: data.modified };
    }
  }
  return null;
}

function ConfirmOverlay({ state, onCancel }: { state: ConfirmState; onCancel: () => void }) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onCancel();
      }
    };
    window.addEventListener('keydown', handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, [onCancel]);

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
      onClick={(e) => { e.stopPropagation(); onCancel(); }}
    >
      <div
        className="w-full max-w-md bg-background border rounded-xl shadow-2xl p-5"
        onClick={(e) => { e.stopPropagation(); }}
      >
        <div className="flex items-start gap-3">
          <div className="h-9 w-9 shrink-0 rounded-full bg-muted flex items-center justify-center">
            <AlertTriangle className="h-5 w-5 text-amber-500" />
          </div>
          <div className="min-w-0">
            <h3 className="text-base font-semibold">{state.title}</h3>
            <p className="text-sm text-muted-foreground mt-1">{state.description}</p>
          </div>
        </div>
        <div className="flex justify-end gap-2 mt-5">
          <Button variant="outline" size="sm" onClick={onCancel}>Cancel</Button>
          <Button
            variant={state.destructive ? "destructive" : "default"}
            size="sm"
            onClick={state.onConfirm}
          >
            {state.confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

export default function TextViewerModal({ file, isOpen, onClose, editable = false, onSaved }: TextViewerModalProps) {
  const [content, setContent] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [language, setLanguage] = useState<string>("text");
  const [isCopied, setIsCopied] = useState(false);
  const [contentLoaded, setContentLoaded] = useState(false);

  // Edit state
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState<string>("");
  const [originalContent, setOriginalContent] = useState<string>("");
  const [baseline, setBaseline] = useState<Baseline>({ size: 0, modified: "" });
  const [effectiveSize, setEffectiveSize] = useState(file?.size ?? 0);
  const [isSaving, setIsSaving] = useState(false);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);

  // Find-in-file state (edit mode)
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [matchCase, setMatchCase] = useState(false);
  const [activeMatch, setActiveMatch] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const highlightRef = useRef<HTMLDivElement>(null);
  const activeMarkRef = useRef<HTMLElement>(null);
  const [mirrorSize, setMirrorSize] = useState<{ width: number; height: number } | null>(null);

  const loadSeqRef = useRef(0);
  const savingRef = useRef(false);

  const { theme } = useTheme();
  // Using vs2015 for dark mode and vs for light mode
  const syntaxStyle = theme === 'dark' ? vs2015 : vs;

  const loadContent = useCallback(async (target: FileInterface): Promise<boolean> => {
    const seq = ++loadSeqRef.current;
    setIsLoading(true);
    setError(null);
    try {
      const response = await axiosLayer.get(target.url, {
        responseType: 'text',
      });
      if (seq !== loadSeqRef.current) return false; // superseded by a newer load
      const text = typeof response.data === 'string' ? response.data : String(response.data ?? "");
      setContent(text);
      setOriginalContent(text);
      setDraft(text);
      setBaseline({ size: target.size, modified: target.modified });
      setEffectiveSize(target.size);
      setContentLoaded(true);
      return true;
    } catch (err: unknown) {
      if (seq !== loadSeqRef.current) return false;
      console.error("Failed to fetch document:", err);
      setError("Failed to load document content. The file might be too large or inaccessible.");
      setContentLoaded(false);
      return false;
    } finally {
      if (seq === loadSeqRef.current) setIsLoading(false);
    }
  }, []);

  // Reset all per-file state whenever the modal opens or switches files.
  useEffect(() => {
    if (!isOpen || !file) return;

    // Invalidate any in-flight fetch for a previously selected file.
    loadSeqRef.current += 1;

    setIsEditing(false);
    setIsSaving(false);
    setConfirm(null);
    setIsCopied(false);
    setSearchOpen(false);
    setSearchQuery("");
    setMatchCase(false);
    setActiveMatch(0);
    setContent("");
    setDraft("");
    setOriginalContent("");
    setBaseline({ size: 0, modified: "" });
    setContentLoaded(false);
    setError(null);
    setEffectiveSize(file.size);
    setLanguage(getLanguageFromExtension(file.name));

    if (file.size > MAX_TEXT_EDIT_SIZE) {
      return; // Too large to preview/edit; show the download prompt.
    }
    void loadContent(file);
  }, [isOpen, file, loadContent]);

  const isDirty = isEditing && draft !== originalContent;

  const requestClose = useCallback(() => {
    if (isSaving) return;
    if (isDirty) {
      setConfirm({
        title: "Discard unsaved changes?",
        description: "Your edits have not been saved. Closing now will discard them.",
        confirmLabel: "Discard",
        destructive: true,
        onConfirm: () => { setConfirm(null); onClose(); },
      });
      return;
    }
    onClose();
  }, [isDirty, isSaving, onClose]);

  const closeSearch = useCallback(() => {
    setSearchOpen(false);
    setSearchQuery("");
    setActiveMatch(0);
    textareaRef.current?.focus();
  }, []);

  // Keep the latest confirm/close handlers available to the single Escape listener
  // without re-binding it on every keystroke.
  const closeStateRef = useRef<{
    confirm: ConfirmState | null;
    isEditing: boolean;
    searchOpen: boolean;
    closeSearch: (() => void) | null;
    requestClose: (() => void) | null;
  }>({
    confirm: null,
    isEditing: false,
    searchOpen: false,
    closeSearch: null,
    requestClose: null,
  });
  closeStateRef.current = { confirm, isEditing, searchOpen, closeSearch, requestClose };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      // The confirm overlay owns Escape while it is open.
      if (closeStateRef.current.confirm) return;
      // Otherwise, Escape closes the find bar before the modal itself.
      if (closeStateRef.current.isEditing && closeStateRef.current.searchOpen) {
        e.stopPropagation();
        closeStateRef.current.closeSearch?.();
        return;
      }
      e.stopPropagation();
      closeStateRef.current.requestClose?.();
    };

    if (isOpen) {
      window.addEventListener('keydown', handleKeyDown, { capture: true });
    }

    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, [isOpen]);

  // Ctrl/Cmd+F opens the find bar while editing.
  useEffect(() => {
    if (!isOpen || !isEditing) return;
    const handleFind = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'f') {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener('keydown', handleFind, { capture: true });
    return () => {
      window.removeEventListener('keydown', handleFind, { capture: true });
    };
  }, [isOpen, isEditing]);

  // Focus the find input when the bar opens.
  useEffect(() => {
    if (isEditing && searchOpen) {
      searchInputRef.current?.focus();
    }
  }, [isEditing, searchOpen]);

  const isTooLarge = effectiveSize > MAX_TEXT_EDIT_SIZE;
  const canEdit = editable && !isTooLarge && !isLoading && !error && contentLoaded;
  const draftBytes = useMemo(() => (isEditing ? getByteSize(draft) : 0), [isEditing, draft]);
  const draftTooLarge = isEditing && draftBytes > MAX_TEXT_EDIT_SIZE;

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
    const nodes: React.ReactNode[] = [];
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

  if (!isOpen || !file) return null;

  const handleCopy = () => {
    navigator.clipboard.writeText(content)
      .then(() => {
        setIsCopied(true);
        toast.success("Content copied to clipboard");
        setTimeout(() => { setIsCopied(false); }, 2000);
      })
      .catch(() => {
        toast.error("Failed to copy content");
      });
  };

  const handleDownload = () => {
    downloadFiles([file.path]);
  };

  const startEditing = () => {
    setDraft(content);
    setOriginalContent(content);
    setSearchOpen(false);
    setSearchQuery("");
    setMatchCase(false);
    setActiveMatch(0);
    setIsEditing(true);
  };

  const cancelEditing = () => {
    if (isSaving) return;
    if (draft === originalContent) {
      setIsEditing(false);
      return;
    }
    setConfirm({
      title: "Discard unsaved changes?",
      description: "Your edits to this file have not been saved.",
      confirmLabel: "Discard changes",
      destructive: true,
      onConfirm: () => {
        setConfirm(null);
        setDraft(originalContent);
        setIsEditing(false);
      },
    });
  };

  const reload = async (freshBaseline: Baseline | null) => {
    setIsEditing(false);
    const ok = await loadContent(file);
    if (ok && freshBaseline) {
      setBaseline(freshBaseline);
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

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
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

  const performSave = async (allowEmpty: boolean) => {
    if (savingRef.current) return;
    savingRef.current = true;
    const snapshot = draft;
    setIsSaving(true);
    try {
      const res = await saveTextFile(file.path, snapshot, {
        allowEmpty,
        baseSize: baseline.size,
        baseModified: baseline.modified,
      });
      setContent(snapshot);
      setOriginalContent(snapshot);
      setBaseline({ size: res.size, modified: res.modified });
      setEffectiveSize(res.size);
      setContentLoaded(true);
      setIsEditing(false);
      toast.success("File saved");
      onSaved?.();
    } catch (err: unknown) {
      const status = getErrorStatus(err);
      if (status === 409) {
        const conflictBaseline = getConflictBaseline(err);
        setConfirm({
          title: "File changed on disk",
          description: "This file was modified after you opened it. Reload the latest content? Your unsaved edits will be lost.",
          confirmLabel: "Reload",
          destructive: true,
          onConfirm: () => {
            setConfirm(null);
            void reload(conflictBaseline);
          },
        });
      } else if (status === 413) {
        toast.error("This file is too large to save.");
      } else {
        toast.error(getErrorMessage(err));
      }
    } finally {
      savingRef.current = false;
      setIsSaving(false);
    }
  };

  const handleSave = () => {
    if (!isEditing || isSaving) return;
    if (!isDirty) {
      setIsEditing(false);
      return;
    }
    if (draft === "") {
      // Safety: never silently wipe a file. Require an explicit confirmation.
      setConfirm({
        title: "Save an empty file?",
        description: "You have removed all text. Saving now will empty this file. This cannot be undone.",
        confirmLabel: "Save empty file",
        destructive: true,
        onConfirm: () => { setConfirm(null); void performSave(true); },
      });
      return;
    }
    if (draftBytes > MAX_TEXT_EDIT_SIZE) {
      setConfirm({
        title: "File is larger than 2 MB",
        description: "This file exceeds the 2 MB preview limit, so it may not be viewable in the built-in viewer after saving. You can still download it. Save anyway?",
        confirmLabel: "Save anyway",
        onConfirm: () => { setConfirm(null); void performSave(false); },
      });
      return;
    }
    void performSave(false);
  };

  return (
    <div
      className={cn(
        "fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm animate-in fade-in duration-200",
        isEditing ? "p-0" : "p-2 sm:p-4",
      )}
      onClick={requestClose}
    >
      <div 
        className={cn(
          "relative flex flex-col bg-background shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200",
          isEditing
            ? "w-full h-full max-w-none border-0 rounded-none"
            : "w-full max-w-5xl h-[90vh] border rounded-xl",
        )}
        onClick={(e) => { e.stopPropagation(); }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30 shrink-0">
          <div className="flex items-center space-x-4 overflow-hidden flex-1">
            <h2 className="text-lg font-semibold whitespace-nowrap overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] min-w-0" title={file.name}>
              {file.name}
            </h2>
            
            {!isTooLarge && !isEditing && (
              <div className="relative hidden sm:flex items-center shrink-0">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm" className="h-8 gap-2 border-input bg-background/50">
                      {SUPPORTED_LANGUAGES.find((l) => l.id === language)?.name ?? "Language"}
                      <ChevronDown className="h-4 w-4 text-muted-foreground" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-[180px] max-h-[300px] overflow-y-auto z-[150]">
                    {SUPPORTED_LANGUAGES.map((lang) => (
                      <DropdownMenuItem 
                        key={lang.id} 
                        onSelect={() => { setLanguage(lang.id); }}
                        className={language === lang.id ? "bg-muted" : ""}
                      >
                        {lang.name}
                        {language === lang.id && <Check className="ml-auto h-4 w-4" />}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            )}
          </div>

          <div className="flex items-center space-x-1 shrink-0 ml-2">
            {isEditing ? (
              <>
                <Button
                  variant={searchOpen ? "secondary" : "ghost"}
                  size="icon"
                  className="h-8 w-8"
                  onClick={toggleSearch}
                  disabled={isSaving}
                  title="Find in file (Ctrl+F)"
                >
                  <Search className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8"
                  onClick={cancelEditing}
                  disabled={isSaving}
                >
                  Cancel
                </Button>
                <Button
                  size="sm"
                  className="h-8 gap-2"
                  onClick={handleSave}
                  disabled={isSaving || !isDirty}
                >
                  {isSaving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Save
                </Button>
              </>
            ) : (
              <>
                {canEdit && (
                  <Button variant="ghost" size="icon" onClick={startEditing} title="Edit file">
                    <Pencil className="h-4 w-4" />
                  </Button>
                )}
                {!isTooLarge && (
                  <div className="sm:hidden relative mr-1">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm" className="h-8 px-2 gap-1 border-input bg-background/50 text-xs">
                          <span className="truncate max-w-[70px]">
                            {SUPPORTED_LANGUAGES.find((l) => l.id === language)?.name ?? "Lang"}
                          </span>
                          <ChevronDown className="h-3 w-3 text-muted-foreground" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-[180px] max-h-[300px] overflow-y-auto z-[150]">
                        {SUPPORTED_LANGUAGES.map((lang) => (
                          <DropdownMenuItem 
                            key={lang.id} 
                            onSelect={() => { setLanguage(lang.id); }}
                            className={language === lang.id ? "bg-muted" : ""}
                          >
                            {lang.name}
                            {language === lang.id && <Check className="ml-auto h-4 w-4" />}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                )}

                {contentLoaded && !error && (
                  <Button variant="ghost" size="icon" onClick={handleCopy} disabled={isLoading} title="Copy content">
                    {isCopied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                  </Button>
                )}
                <Button variant="ghost" size="icon" onClick={handleDownload} title="Download file">
                  <Download className="h-4 w-4" />
                </Button>
              </>
            )}
            <div className="w-px h-6 bg-border mx-1" />
            <Button variant="ghost" size="icon" onClick={requestClose} disabled={isSaving} title="Close">
              <X className="h-5 w-5" />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-hidden relative bg-background">
          {isTooLarge && !contentLoaded ? (
            <div className="absolute inset-0 flex flex-col items-center justify-center p-6 text-center text-muted-foreground">
              <div className="h-16 w-16 bg-muted rounded-full flex items-center justify-center mb-4">
                <FileText className="h-8 w-8" />
              </div>
              <p className="font-semibold text-lg mb-2 text-foreground">File is too large</p>
              <p className="mb-6 max-w-sm">
                This file is larger than 2MB and cannot be previewed in the browser. Please download it to view its contents.
              </p>
              <Button onClick={handleDownload} className="gap-2">
                <Download className="h-4 w-4" />
                Download File
              </Button>
            </div>
          ) : isLoading ? (
            <div className="absolute inset-0 flex flex-col items-center justify-center text-muted-foreground">
              <div className="h-8 w-8 rounded-full border-4 border-primary border-t-transparent animate-spin mb-4" />
              <p>Loading document...</p>
            </div>
          ) : error ? (
            <div className="absolute inset-0 flex flex-col items-center justify-center text-red-500 p-6 text-center">
              <p className="font-semibold text-lg mb-2">Error</p>
              <p>{error}</p>
            </div>
          ) : isEditing ? (
            <div className="h-full w-full flex flex-col">
              {searchOpen && (
                <div className="shrink-0 flex items-center gap-1.5 px-3 py-2 border-b bg-muted/20">
                  <Search className="h-4 w-4 text-muted-foreground shrink-0" />
                  <input
                    ref={searchInputRef}
                    value={searchQuery}
                    onChange={(e) => { setSearchQuery(e.target.value); setActiveMatch(-1); }}
                    onKeyDown={handleSearchKeyDown}
                    spellCheck={false}
                    placeholder="Find in file"
                    className="flex-1 min-w-0 h-7 px-2 rounded-md border border-input bg-background text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  />
                  <span className="text-xs text-muted-foreground tabular-nums min-w-[64px] text-right shrink-0">
                    {searchQuery === ""
                      ? ""
                      : matches.length === 0
                        ? "No results"
                        : `${String(highlightActive < 0 ? 1 : highlightActive + 1)}/${String(matches.length)}`}
                  </span>
                  <Button
                    variant={matchCase ? "secondary" : "ghost"}
                    size="icon"
                    className="h-7 w-7 shrink-0 text-xs font-semibold"
                    onClick={() => { setMatchCase((prev) => !prev); setActiveMatch(-1); }}
                    title="Match case"
                  >
                    Aa
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={findPrev}
                    disabled={matches.length === 0}
                    title="Previous match (Shift+Enter)"
                  >
                    <ChevronUp className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={findNext}
                    disabled={matches.length === 0}
                    title="Next match (Enter)"
                  >
                    <ChevronDown className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={closeSearch}
                    title="Close find"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              )}
              <div className="relative flex-1 min-h-0 overflow-hidden">
                {mirrorEnabled && (
                  <div
                    ref={highlightRef}
                    aria-hidden="true"
                    className="absolute left-0 top-0 overflow-hidden pointer-events-none select-none whitespace-pre-wrap break-words p-4 font-mono text-sm leading-relaxed text-transparent"
                    style={mirrorSize ? { width: mirrorSize.width, height: mirrorSize.height } : undefined}
                  >
                    {highlightedContent}
                  </div>
                )}
                <textarea
                  ref={textareaRef}
                  value={draft}
                  onChange={(e) => { setDraft(e.target.value); }}
                  onScroll={syncHighlightScroll}
                  spellCheck={false}
                  className="absolute inset-0 h-full w-full resize-none outline-none overflow-auto whitespace-pre-wrap break-words p-4 font-mono text-sm leading-relaxed bg-transparent text-foreground"
                  placeholder="File is empty"
                />
              </div>
              <div className="shrink-0 flex items-center justify-between gap-2 px-4 py-2 border-t text-xs text-muted-foreground">
                <span className={draftTooLarge ? "text-amber-600 dark:text-amber-400" : ""}>
                  {draftTooLarge
                    ? "Larger than 2 MB — it may not be previewable after saving."
                    : "Editing"}
                </span>
                <span>{formatBytes(draftBytes)}</span>
              </div>
            </div>
          ) : (
            <div className="h-full w-full flex flex-col">
              {isTooLarge && (
                <div className="shrink-0 px-4 py-2 border-b bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs">
                  Saved, but this file is larger than 2 MB and won't be previewable the next time it is opened.
                </div>
              )}
              <div className="flex-1 overflow-auto text-sm">
                {language === 'text' ? (
                  <pre className="p-4 whitespace-pre-wrap font-mono text-foreground break-words min-h-full">
                    {content}
                  </pre>
                ) : (
                  <SyntaxHighlighter
                    language={language}
                    style={syntaxStyle}
                    customStyle={{ margin: 0, padding: '1rem', minHeight: '100%', background: 'transparent' }}
                    showLineNumbers={true}
                    wrapLines={true}
                    lineNumberStyle={{ userSelect: "none", WebkitUserSelect: "none", MozUserSelect: "none", msUserSelect: "none" }}
                  >
                    {content}
                  </SyntaxHighlighter>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {confirm && <ConfirmOverlay state={confirm} onCancel={() => { setConfirm(null); }} />}
    </div>
  );
}
