import type { Baseline } from "./types";

export const SUPPORTED_LANGUAGES = [
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

export function getLanguageFromExtension(filename: string): string {
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
export function getByteSize(value: string): number {
  return new Blob([value]).size;
}

/** All non-overlapping start indices of `query` within `text`. */
export function findAllMatches(text: string, query: string, matchCase: boolean): number[] {
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
export const HIGHLIGHT_MAX_MATCHES = 2000;
export const HIGHLIGHT_MAX_BYTES = 512 * 1024;

export function getErrorStatus(err: unknown): number | undefined {
  if (err && typeof err === "object" && "response" in err) {
    return (err as { response?: { status?: number } }).response?.status;
  }
  return undefined;
}

export function getErrorMessage(err: unknown): string {
  if (err && typeof err === "object" && "response" in err) {
    const data = (err as { response?: { data?: { error?: string } } }).response?.data;
    if (data && typeof data.error === "string" && data.error) return data.error;
  }
  return "Failed to save file. Please try again.";
}

/** Extract the on-disk state the backend included with a 409 conflict. */
export function getConflictBaseline(err: unknown): Baseline | null {
  if (err && typeof err === "object" && "response" in err) {
    const data = (err as { response?: { data?: { data?: { size?: unknown; modified?: unknown } } } })
      .response?.data?.data;
    if (data && typeof data.size === "number" && typeof data.modified === "string") {
      return { size: data.size, modified: data.modified };
    }
  }
  return null;
}
