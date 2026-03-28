import { useEffect, useRef } from "react";
import { X, GripVertical, Shuffle, ListMusic, Play } from "lucide-react";
import { type FileInterface } from "@/api/api-file";
import { Button } from "@/components/ui/button";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

interface MusicQueueModalProps {
  isOpen: boolean;
  onClose: () => void;
  playlist: FileInterface[];
  currentFile: FileInterface | null;
  onSelectMusic: (file: FileInterface) => void;
  onReorderPlaylist: (newPlaylist: FileInterface[]) => void;
}

const SortableItem = ({
  item,
  isActive,
  onSelect,
  activeRef,
}: {
  item: FileInterface;
  isActive: boolean;
  onSelect: (file: FileInterface) => void;
  activeRef?: React.Ref<HTMLDivElement>;
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.url });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 10 : 1,
  };

  // Combine refs so both dnd-kit and our scroll logic can access the DOM node
  const setCombinedRef = (node: HTMLDivElement | null) => {
    setNodeRef(node);
    if (activeRef) {
      if (typeof activeRef === 'function') {
        activeRef(node);
      } else {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-member-access
        (activeRef as any).current = node;
      }
    }
  };

  return (
    <div
      ref={setCombinedRef}
      style={style}
      className={`flex items-center justify-between p-2 mb-1 rounded-md transition-colors ${
        isActive ? "bg-primary/10 text-primary" : "hover:bg-muted/50"
      } ${isDragging ? "opacity-50" : "opacity-100"}`}
    >
      <div 
        className="flex-1 min-w-0 flex items-center gap-3 cursor-pointer"
        onClick={() => { onSelect(item); }}
      >
        <div className="w-6 flex justify-center shrink-0">
          {isActive ? (
            <Play className="w-4 h-4 text-primary fill-primary" />
          ) : (
            <span className="text-muted-foreground text-xs">{/* Empty placeholder or number could go here */}</span>
          )}
        </div>
        <span className="truncate text-sm font-medium">{item.name}</span>
      </div>
      
      {/* Drag Handle */}
      <div
        {...attributes}
        {...listeners}
        className="p-2 cursor-grab active:cursor-grabbing text-muted-foreground hover:text-foreground shrink-0 touch-none"
      >
        <GripVertical className="w-4 h-4" />
      </div>
    </div>
  );
};

export function MusicQueueModal({
  onClose,
  playlist,
  currentFile,
  onSelectMusic,
  onReorderPlaylist,
  isOpen,
}: MusicQueueModalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const activeItemRef = useRef<HTMLDivElement>(null);
  const hasScrolledRef = useRef(false);

  // Scroll to active item when modal opens
  useEffect(() => {
    if (isOpen) {
      hasScrolledRef.current = false;
      // Use a slight delay to allow rendering before scroll
      const timer = setTimeout(() => {
        if (activeItemRef.current && containerRef.current && !hasScrolledRef.current) {
          activeItemRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
          hasScrolledRef.current = true;
        }
      }, 100);
      return () => { clearTimeout(timer); };
    }
  }, [isOpen]);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5, // Start dragging after moving 5 pixels
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = playlist.findIndex((item) => item.url === active.id);
      const newIndex = playlist.findIndex((item) => item.url === over.id);
      onReorderPlaylist(arrayMove(playlist, oldIndex, newIndex));
    }
  };

  const handleShuffle = () => {
    const shuffled = [...playlist];
    // Fisher-Yates shuffle
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    // Automatically move the current file to the top on shuffle
    if (currentFile) {
      const currentIndex = shuffled.findIndex(f => f.url === currentFile.url);
      if (currentIndex > 0) {
        const [curr] = shuffled.splice(currentIndex, 1);
        shuffled.unshift(curr);
      }
    }
    onReorderPlaylist(shuffled);
    
    // Scroll back to top (where the active item now is)
    if (containerRef.current) {
      containerRef.current.scrollTo({ top: 0, behavior: 'smooth' });
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b shrink-0">
        <div className="flex items-center gap-2 font-semibold">
          <ListMusic className="w-5 h-5" />
          <h2>Playing Next</h2>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={handleShuffle} className="h-8 text-xs gap-1">
            <Shuffle className="w-3.5 h-3.5" />
            Shuffle
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full md:hidden" onClick={onClose}>
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Playlist */}
      <div 
        ref={containerRef}
        className="flex-1 overflow-y-auto p-2 scrollbar-thin scrollbar-thumb-muted-foreground/20 scrollbar-track-transparent"
      >
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={playlist.map(item => item.url)} strategy={verticalListSortingStrategy}>
            <div className="flex flex-col pb-safe">
              {playlist.map((item) => (
                <SortableItem
                  key={item.url}
                  item={item}
                  isActive={currentFile?.url === item.url}
                  onSelect={onSelectMusic}
                  activeRef={currentFile?.url === item.url ? activeItemRef : undefined}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      </div>
    </div>
  );
}
