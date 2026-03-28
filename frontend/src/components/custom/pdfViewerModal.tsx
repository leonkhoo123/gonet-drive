import { useEffect, useState, useRef } from "react";
import { X, Download, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, Loader2 } from "lucide-react";
import { type FileInterface, downloadFiles } from "@/api/api-file";
import { Button } from "@/components/ui/button";
import { Document, Page, pdfjs } from 'react-pdf';
import { TransformWrapper, TransformComponent } from "react-zoom-pan-pinch";
import 'react-pdf/dist/Page/AnnotationLayer.css';
import 'react-pdf/dist/Page/TextLayer.css';
import axiosLayer from "@/api/axiosLayer";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString();

interface PdfViewerModalProps {
  file: FileInterface | null;
  isOpen: boolean;
  onClose: () => void;
}

export default function PdfViewerModal({ file, isOpen, onClose }: PdfViewerModalProps) {
  const [numPages, setNumPages] = useState<number>();
  const [pageNumber, setPageNumber] = useState<number>(1);
  const [scale, setScale] = useState<number>(1.0);
  const [containerWidth, setContainerWidth] = useState<number>();
  const containerRef = useRef<HTMLDivElement>(null);
  
  const [pdfData, setPdfData] = useState<string | null>(null);
  const [isLoadingFile, setIsLoadingFile] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [showControls, setShowControls] = useState(true);

  const isTooLarge = file && file.size > 20 * 1024 * 1024; // 20MB limit

  // Fetch the PDF using axiosLayer to handle authentication/refresh
  useEffect(() => {
    if (!isOpen || !file) {
      setPdfData(null);
      return;
    }

    if (isTooLarge) {
      setFetchError("File is too large to preview in the browser. Please download it instead.");
      return;
    }

    let isMounted = true;
    let objectUrl: string | null = null;
    const fetchPdf = async () => {
      setIsLoadingFile(true);
      setFetchError(null);
      try {
        const response = await axiosLayer.get(file.url, { responseType: 'blob' });
        if (isMounted) {
          const blob = new Blob([response.data as BlobPart], { type: "application/pdf" });
          objectUrl = URL.createObjectURL(blob);
          setPdfData(objectUrl);
        }
      } catch (err) {
        console.error("Error fetching PDF:", err);
        if (isMounted) {
          setFetchError("Failed to load PDF file. It might be inaccessible or you need to login again.");
        }
      } finally {
        if (isMounted) {
          setIsLoadingFile(false);
        }
      }
    };
    
    void fetchPdf();
    
    return () => {
      isMounted = false;
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [isOpen, file]);

  // Reset state when file changes
  useEffect(() => {
    if (file) {
      setPageNumber(1);
      setScale(1.0);
      setNumPages(undefined);
      setShowControls(true);
    }
  }, [file]);

  useEffect(() => {
    if (!isOpen) return;

    const updateWidth = () => {
      if (containerRef.current) {
        setContainerWidth(containerRef.current.clientWidth);
      }
    };

    // Delay slightly to ensure container is fully rendered
    const timeoutId = setTimeout(updateWidth, 50);

    window.addEventListener('resize', updateWidth);
    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener('resize', updateWidth);
    };
  }, [isOpen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      } else if (e.key === 'ArrowRight' && numPages && pageNumber < numPages) {
        setPageNumber(prev => Math.min(numPages, prev + 1));
      } else if (e.key === 'ArrowLeft' && pageNumber > 1) {
        setPageNumber(prev => Math.max(1, prev - 1));
      }
    };

    if (isOpen) {
      window.addEventListener('keydown', handleKeyDown, { capture: true });
    }

    return () => {
      window.removeEventListener('keydown', handleKeyDown, { capture: true });
    };
  }, [isOpen, onClose, numPages, pageNumber]);

  if (!isOpen || !file) return null;

  const handleDownload = () => {
    downloadFiles([file.path]);
  };

  function onDocumentLoadSuccess({ numPages }: { numPages: number }): void {
    setNumPages(numPages);
    setPageNumber(1);
  }

  // Calculate width for mobile responsiveness
  const getPageWidth = () => {
    if (containerWidth) {
      // 32px for padding (16px * 2)
      return Math.min(containerWidth - 32, 1000) * scale;
    }
    return undefined;
  };

  const toggleControls = (e: React.MouseEvent) => {
    // Prevent toggling if the click originated from a control button
    if ((e.target as HTMLElement).closest('button')) {
      return;
    }
    
    // Check if user is selecting text - don't toggle if they are
    const selection = window.getSelection();
    if (selection && selection.toString().length > 0) {
      return;
    }
    
    setShowControls(prev => !prev);
  };

  return (
    <div className="fixed inset-0 z-[100] flex flex-col w-screen h-screen bg-background/95 backdrop-blur-sm animate-in fade-in duration-200 text-foreground">
      {/* Header */}
      <div className={`absolute top-0 left-0 w-full z-50 flex items-center justify-between px-4 py-3 border-b bg-background/80 backdrop-blur-sm transition-all duration-300 ease-in-out ${showControls ? 'translate-y-0 opacity-100' : '-translate-y-full opacity-0'}`}>
        <div className="flex items-center space-x-4 overflow-hidden flex-1">
          <h2 className="text-lg font-semibold whitespace-nowrap overflow-x-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] min-w-0" title={file.name}>
            {file.name}
          </h2>
        </div>
        
        <div className="flex items-center gap-2 shrink-0 ml-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleDownload}
            className="h-8 w-8 rounded-full"
            title="Download"
          >
            <Download className="h-5 w-5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="h-8 w-8 rounded-full"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>
      </div>

      {/* Content - PDF Document */}
      <div 
        ref={containerRef}
        className="flex-1 w-full h-full overflow-hidden flex flex-col items-center relative"
      >
        {isLoadingFile ? (
          <div className="flex flex-col items-center justify-center h-full w-full gap-4">
             <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
             <span className="text-muted-foreground animate-pulse">Downloading PDF...</span>
          </div>
        ) : fetchError ? (
          <div className="flex flex-col items-center justify-center h-full w-full text-destructive gap-4">
            <span>{fetchError}</span>
            <Button variant="outline" onClick={handleDownload}>
              <Download className="h-4 w-4 mr-2" /> Download Instead
            </Button>
          </div>
        ) : pdfData ? (
          <TransformWrapper
            initialScale={1}
            minScale={0.5}
            maxScale={5}
            centerOnInit={true}
            centerZoomedOut={true}
            wheel={{ step: 0.1 }}
            pinch={{ step: 5 }}
            doubleClick={{ disabled: true }}
            panning={{ velocityDisabled: true }}
          >
            {({ zoomIn, zoomOut, resetTransform, centerView }) => (
              <>
                {/* Controls Bar - Floating at bottom */}
                <div 
                  className={`fixed bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-2 sm:gap-4 px-2 sm:px-4 py-2 bg-background/80 border rounded-full shadow-lg z-50 backdrop-blur-sm transition-all duration-300 ${showControls ? 'translate-y-0 opacity-100' : 'translate-y-20 opacity-0 pointer-events-none'}`}
                  onClick={(e) => { e.stopPropagation(); }}
                >
                  {/* Pagination (Left) */}
                  <div className="flex items-center gap-1 sm:gap-2">
                    {numPages ? (
                      <>
                        <Button 
                          variant="ghost" 
                          size="icon"
                          className="h-8 w-8 rounded-full"
                          onClick={(e) => {
                            e.stopPropagation();
                            setPageNumber(p => Math.max(1, p - 1));
                          }}
                          disabled={pageNumber <= 1}
                        >
                          <ChevronLeft className="h-4 w-4" />
                        </Button>
                        <span className="text-sm font-medium whitespace-nowrap min-w-[3rem] text-center">
                          {pageNumber} / {numPages}
                        </span>
                        <Button 
                          variant="ghost" 
                          size="icon"
                          className="h-8 w-8 rounded-full"
                          onClick={(e) => {
                            e.stopPropagation();
                            setPageNumber(p => Math.min(numPages, p + 1));
                          }}
                          disabled={pageNumber >= numPages}
                        >
                          <ChevronRight className="h-4 w-4" />
                        </Button>
                      </>
                    ) : <div className="w-24" />}
                  </div>

                  <div className="w-px h-6 bg-border mx-1 hidden sm:block"></div>

                  {/* Zoom Controls (Right) */}
                  <div className="flex items-center gap-1">
                     <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full" onClick={(e) => { e.stopPropagation(); zoomOut(); }}>
                       <ZoomOut className="h-4 w-4" />
                     </Button>
                     <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full hidden sm:flex" onClick={(e) => { e.stopPropagation(); resetTransform(); }}>
                       <span className="text-xs font-medium">100%</span>
                     </Button>
                     <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full" onClick={(e) => { e.stopPropagation(); zoomIn(); }}>
                       <ZoomIn className="h-4 w-4" />
                     </Button>
                  </div>
                </div>

                <TransformComponent wrapperClass="!w-full !h-full" contentClass="flex justify-center items-center">
                  <div 
                    className="flex flex-col items-center justify-center cursor-default pb-16 pt-16 sm:py-16 px-2 sm:px-4"
                    onClick={toggleControls}
                  >
                    <Document
                      file={pdfData}
                      onLoadSuccess={onDocumentLoadSuccess}
                      loading={
                        <div className="flex items-center justify-center h-[50vh] w-full max-w-2xl bg-white/50 rounded-lg">
                          <span className="text-muted-foreground animate-pulse">Rendering PDF...</span>
                        </div>
                      }
                      error={
                        <div className="flex flex-col items-center justify-center h-[50vh] w-full max-w-2xl bg-white/50 rounded-lg text-destructive gap-4">
                          <span>Failed to render PDF file.</span>
                          <Button variant="outline" onClick={(e) => { e.stopPropagation(); handleDownload(); }}>
                            <Download className="h-4 w-4 mr-2" /> Download Instead
                          </Button>
                        </div>
                      }
                      className="flex flex-col items-center w-fit"
                    >
                      {numPages ? (
                        <div className="mb-2 transition-opacity duration-300">
                          <Page 
                            pageNumber={pageNumber} 
                            scale={1.0} 
                            width={getPageWidth()}
                            renderTextLayer={true}
                            renderAnnotationLayer={true}
                            className="shadow-xl bg-white max-w-full"
                            onLoadSuccess={() => {
                              setTimeout(() => {
                                centerView();
                              }, 10);
                            }}
                            loading={
                              <div className="flex items-center justify-center w-full min-h-[50vh] bg-white shadow-xl">
                                 <span className="text-muted-foreground animate-pulse">Loading page...</span>
                              </div>
                            }
                          />
                        </div>
                      ) : (
                        <div className="flex items-center justify-center w-full min-h-[50vh] max-w-2xl bg-white shadow-xl">
                           <span className="text-muted-foreground animate-pulse">Loading pages...</span>
                        </div>
                      )}
                    </Document>
                  </div>
                </TransformComponent>
              </>
            )}
          </TransformWrapper>
        ) : null}
      </div>
    </div>
  );
}
