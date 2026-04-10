import { useState, useEffect } from "react";
import { X, Download, Copy, Check, ChevronDown, FileText } from "lucide-react";
import { type FileInterface, downloadFiles } from "@/api/api-file";
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

export default function TextViewerModal({ file, isOpen, onClose }: TextViewerModalProps) {
  const [content, setContent] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [language, setLanguage] = useState<string>("text");
  const [isCopied, setIsCopied] = useState(false);

  // Handle escape key to close modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    };

    if (isOpen) {
      window.addEventListener('keydown', handleKeyDown, { capture: true });
    }

    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, [isOpen, onClose]);
  
  const { theme } = useTheme();
  // Using vs2015 for dark mode and vs for light mode
  const syntaxStyle = theme === 'dark' ? vs2015 : vs;

  useEffect(() => {
    if (!isOpen || !file) return;

    if (file.size > 2 * 1024 * 1024) {
      return; // Skip fetching if file is too large
    }

    const fetchContent = async () => {
      setIsLoading(true);
      setError(null);
      setContent("");
      
      // Auto-detect language
      setLanguage(getLanguageFromExtension(file.name));

      try {
        const response = await axiosLayer.get(file.url, {
          responseType: 'text',
        });
        setContent(response.data as string);
      } catch (err: any) {
        console.error("Failed to fetch document:", err);
        setError("Failed to load document content. The file might be too large or inaccessible.");
      } finally {
        setIsLoading(false);
      }
    };

    void fetchContent();
  }, [isOpen, file]);

  if (!isOpen || !file) return null;

  const isTooLarge = file.size > 2 * 1024 * 1024;

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

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-2 sm:p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
      <div 
        className="relative flex flex-col w-full max-w-5xl h-[90vh] bg-background border rounded-xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200"
        onClick={(e) => { e.stopPropagation(); }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30 shrink-0">
          <div className="flex items-center space-x-4 overflow-hidden flex-1">
            <h2 className="text-lg font-semibold whitespace-nowrap overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] min-w-0" title={file.name}>
              {file.name}
            </h2>
            
            {!isTooLarge && (
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

            {!isTooLarge && (
              <Button variant="ghost" size="icon" onClick={handleCopy} disabled={isLoading || !!error} title="Copy content">
                {isCopied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
              </Button>
            )}
            {!isTooLarge && (
              <Button variant="ghost" size="icon" onClick={handleDownload} title="Download file">
                <Download className="h-4 w-4" />
              </Button>
            )}
            <div className="w-px h-6 bg-border mx-1" />
            <Button variant="ghost" size="icon" onClick={onClose} title="Close">
              <X className="h-5 w-5" />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-hidden relative bg-background">
          {file.size > 2 * 1024 * 1024 ? (
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
          ) : (
            <div className="h-full w-full overflow-auto text-sm">
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
          )}
        </div>
      </div>
    </div>
  );
}
