import { useState, useEffect } from "react";
import { Folder, BookAudio, ArrowLeft, RefreshCcw, PlayCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getAudioBookFileList } from "@/api/api-audiobook";
import type { AudioBookItem } from "@/api/api-audiobook";
import { formatBytes } from "@/utils/utils";
import { toast } from "sonner";
import DefaultLayout from "@/layouts/DefaultLayout";
import { AudioBookPlayer } from "@/components/custom/audioBookPlayer";
import { TruncatedText } from "@/components/custom/truncatedText";
import { useNavigate } from "react-router-dom";

function formatDuration(seconds: number): string {
  if (!seconds) return "0 min";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  
  // Build the string parts based on whether hours, minutes, and seconds are present.
  const parts = [];
  if (h > 0) parts.push(`${h} hr`);
  if (m > 0 || h > 0) parts.push(`${m} min`);
  
  return parts.join(' ');
}

export default function AudioBookPage() {
  const navigate = useNavigate();
  
  // Library State
  const [items, setItems] = useState<AudioBookItem[]>([]);
  const [currentPath, setCurrentPath] = useState("/.audio_book");
  const [loading, setLoading] = useState(false);
  
  // Player State
  const [currentBook, setCurrentBook] = useState<AudioBookItem | null>(null);

  useEffect(() => {
    void fetchData();
  }, [currentPath]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const data = await getAudioBookFileList(currentPath);
      setItems(data.items);
    } catch {
      toast.error("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleItemClick = (item: AudioBookItem) => {
    if (item.mediaType === "dir") {
      setCurrentPath(currentPath === "/" ? `/${item.name}` : `${currentPath}/${item.name}`);
    } else {
      setCurrentBook(item);
    }
  };

  const handleBack = () => {
    if (currentPath === "/.audio_book" || currentPath === "/") {
      void navigate("/home");
      return;
    }
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    setCurrentPath(parts.length > 0 ? "/" + parts.join("/") : "/.audio_book");
  };

  const currentFolderName = currentPath === "/.audio_book" || currentPath === "/" 
    ? "Audio Book" 
    : currentPath.split("/").filter(Boolean).pop();

  return (
    <DefaultLayout>
      <div className="flex flex-col h-full bg-background relative pb-24">
        {/* Top Breadcrumb / Navigation Bar */}
        <div className="h-16 md:h-14 border-b flex items-center justify-between px-2 md:px-6 bg-background shrink-0 gap-2">
          <div className="flex items-center gap-2 overflow-hidden">
            <div className="flex items-center text-sm text-muted-foreground overflow-hidden whitespace-nowrap">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { void navigate("/home"); }}
                className="p-1 h-auto bg-transparent hover:bg-transparent hover:underline text-foreground shrink-0 hidden md:inline-flex"
              >
                Home
              </Button>
              <div className="hidden md:flex items-center">
                <span className="flex items-center shrink-0">
                  <span className="mx-1 text-muted-foreground/50">/</span>
                  <button
                    onClick={() => { setCurrentPath("/.audio_book"); }}
                    className={`hover:underline hover:text-foreground transition-colors ${currentPath === "/.audio_book" ? "font-medium text-foreground" : ""}`}
                  >
                    Audio Book
                  </button>
                </span>
                {currentPath !== "/.audio_book" && currentPath !== "/" && currentPath.split("/").filter(Boolean).slice(1).map((part, idx, arr) => {
                  return (
                    <span key={idx} className="flex items-center shrink-0">
                      <span className="mx-1 text-muted-foreground/50">/</span>
                      <button
                        onClick={() => {
                          const targetPath = "/.audio_book/" + arr.slice(0, idx + 1).join("/");
                          setCurrentPath(targetPath);
                        }}
                        className={`hover:underline hover:text-foreground transition-colors ${idx === arr.length - 1 ? "font-medium text-foreground" : ""}`}
                      >
                        {part}
                      </button>
                    </span>
                  );
                })}
              </div>
              <div className="md:hidden flex items-center shrink-0">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handleBack}
                  className="mr-1 h-12 w-12 md:h-8 md:w-8 text-muted-foreground hover:text-foreground shrink-0"
                >
                  <ArrowLeft className="h-8 w-8 md:h-5 md:w-5" />
                </Button>
                <span className="font-semibold text-foreground text-lg md:text-base">
                  {currentFolderName}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {/* Mobile Action Buttons */}
            <div className="md:hidden flex items-center">
              <Button variant="ghost" size="icon" className="h-12 w-12 text-muted-foreground" onClick={() => void fetchData()}>
                <RefreshCcw className="h-6 w-6" />
                <span className="sr-only">Refresh</span>
              </Button>
            </div>
            {/* Desktop Action Buttons */}
            <div className="hidden md:flex items-center">
              <Button variant="ghost" size="icon" className="h-9 w-9 text-muted-foreground" onClick={() => void fetchData()}>
                <RefreshCcw className="h-5 w-5" />
                <span className="sr-only">Refresh</span>
              </Button>
            </div>
          </div>
        </div>

        {/* Desktop Toolbar (Hidden on Mobile) */}
        <div className="hidden md:flex items-center gap-2 px-6 py-2 border-b bg-muted/30 shrink-0 h-14">
          <Button variant="ghost" size="sm" onClick={handleBack} className="text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
        </div>

        {/* Desktop Column Headers */}
        <div className="hidden md:flex border-b font-semibold py-2 px-6 md:pl-6 md:pr-8 text-sm bg-muted/30 shrink-0">
          <div className="flex-1 text-left text-muted-foreground">Name</div>
          <div className="w-32 hidden lg:flex justify-end text-muted-foreground">Duration</div>
          <div className="w-32 hidden lg:flex justify-end text-muted-foreground">Size</div>
          <div className="w-32 hidden lg:flex justify-end text-muted-foreground">Progress</div>
        </div>

        {/* Main Content */}
        <div className="flex-1 overflow-auto bg-background">
          <div className="flex flex-col h-full min-h-0">
            {/* List */}
            <div className="flex-1 overflow-auto p-2 md:p-3">
              {loading ? (
                <div className="flex justify-center p-8 text-muted-foreground">Loading...</div>
              ) : items.length === 0 ? (
                <div className="flex flex-col items-center justify-center p-12 text-muted-foreground border rounded-lg bg-card/50 mx-2 md:mx-4 mt-4">
                  <Folder className="h-12 w-12 mb-4 opacity-20" />
                  <p>This folder is empty.</p>
                </div>
              ) : (
                <div className="flex flex-col space-y-1 w-full">
                  {items.map((item) => {
                    const isDir = item.mediaType === "dir";
                    const displayName = !isDir && item.name.startsWith("ab_") ? item.name.slice(3) : item.name;
                    const isSelected = currentBook?.name === item.name;
                    const percent = !isDir && item.total_length > 0 ? Math.min(100, Math.max(0, (item.progress_time / item.total_length) * 100)) : 0;

                    return (
                      <div
                        key={item.name}
                        onClick={() => { handleItemClick(item); }}
                        className={`group relative flex items-center md:px-5 pl-4 pr-2 py-3 md:py-2 cursor-pointer rounded-lg md:rounded-md transition-all duration-75 select-none overflow-hidden border
                          ${isSelected ? 'bg-primary/10 border-primary/20 hover:bg-primary/20' : 'bg-card md:bg-transparent border-border/50 md:border-transparent hover:bg-muted/50'}
                        `}
                      >
                        {/* Progress Background */}
                        {!isDir && percent > 0 && (
                          <div 
                            className="absolute left-0 top-0 bottom-0 bg-primary/10 dark:bg-primary/20 pointer-events-none transition-all md:hidden"
                            style={{ width: `${percent}%` }}
                          />
                        )}

                        {/* Content */}
                        <div className="relative z-10 flex flex-1 items-center w-full min-w-0">
                          {/* Name Column (takes remaining space) */}
                          <div className="flex-1 flex items-center space-x-3 min-w-0 pr-4">
                            {isDir ? (
                              <Folder className="h-7 w-7 md:h-5 md:w-5 shrink-0 text-primary fill-primary/20" />
                            ) : (
                              <BookAudio className="h-7 w-7 md:h-5 md:w-5 shrink-0 text-pink-400" />
                            )}
                            <div className="flex flex-col min-w-0 w-full text-left">
                              <div className="flex items-center gap-2">
                                <TruncatedText className="font-medium text-base md:text-sm text-foreground" text={displayName} />
                                {isSelected && !isDir && <PlayCircle className="h-4 w-4 text-primary shrink-0 hidden md:block" />}
                              </div>
                              <div className="text-[13px] md:text-xs mt-0.5 text-muted-foreground flex gap-2 md:hidden">
                                {!isDir && (
                                  <>
                                    <span>{formatDuration(item.total_length)}</span>
                                    <span>•</span>
                                    <span>{formatBytes(item.size)}</span>
                                    {percent > 0 && (
                                      <>
                                        <span>•</span>
                                        <span className="text-primary">{Math.round(percent)}%</span>
                                      </>
                                    )}
                                  </>
                                )}
                                {isDir && <span>Folder</span>}
                              </div>
                            </div>
                          </div>

                          {/* Desktop Only Columns */}
                          <div className="hidden lg:flex w-32 justify-end items-center text-sm text-muted-foreground shrink-0 font-medium">
                            {!isDir ? formatDuration(item.total_length) : "--"}
                          </div>
                          
                          <div className="hidden lg:flex w-32 justify-end items-center text-sm text-muted-foreground shrink-0">
                            {!isDir ? formatBytes(item.size) : "--"}
                          </div>
                          
                          <div className="hidden lg:flex w-32 justify-end items-center shrink-0 pr-2">
                            {!isDir ? (
                              <div className="flex items-center gap-2 w-full justify-end">
                                <span className={`text-sm ${percent > 0 ? 'text-primary font-medium' : 'text-muted-foreground'}`}>
                                  {Math.round(percent)}%
                                </span>
                                <div className="h-1.5 w-16 bg-muted rounded-full overflow-hidden shrink-0">
                                  <div 
                                    className="h-full bg-primary rounded-full transition-all"
                                    style={{ width: `${percent}%` }}
                                  />
                                </div>
                              </div>
                            ) : "--"}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Player */}
        <div className="fixed bottom-0 left-0 right-0 z-[60] pointer-events-none">
          <div className="pointer-events-auto">
            <AudioBookPlayer 
              file={currentBook} 
              onClose={() => { setCurrentBook(null); }} 
            />
          </div>
        </div>
      </div>
    </DefaultLayout>
  );
}
