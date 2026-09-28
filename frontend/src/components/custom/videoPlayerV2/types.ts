import type { RefObject, TouchEvent } from "react";
import type { FileInterface } from "@/api/api-file";
import type { AutoPlayMode } from "@/hooks/useVideoPlayerV2/useVideoAutoPlay";
import type { VideoQuality } from "@/hooks/useVideoPlayerV2/useVideoPlaybackController";
import type { EventSpan } from "@/utils/videoPlayerV2Events";

export interface VideoPlayerModalProps {
  file: FileInterface;
  isOpen: boolean;
  onClose: (
    isDisqualified: boolean,
    oriPath: string,
    isNewName: boolean,
    newName: string,
    rotation: number
  ) => void;
  /** Video files of the current folder, in list order (for auto-play). */
  videoFiles?: FileInterface[];
  /** Swap the open clip without closing the player (auto-play). */
  onSelectVideo?: (file: FileInterface) => void;
  /**
   * Called after an in-player mutation (event save / terminal rename-done) has
   * been queued, so the host can refresh its listing immediately rather than
   * waiting for the async job's WebSocket completion.
   */
  onVideoMutation?: () => void | Promise<void>;
}

export interface VideoSurfaceProps {
  videoRef: RefObject<HTMLVideoElement | null>;
  rotation: number;
  onTap: () => void;
  onTouchStart: (e: TouchEvent<HTMLDivElement>) => void;
  onTouchMove: (e: TouchEvent<HTMLDivElement>) => void;
  onTouchEnd: () => void;
}

export interface VideoDragDeltaProps {
  /** Signed offset from the seek origin, in seconds. */
  seconds: number;
}

export interface VideoTimeDisplayProps {
  currentTime: number;
  duration: number;
  disqualified: boolean;
  isNewName: boolean;
  newName: string;
  fileName: string;
  isRotation: boolean;
  rotation: number;
}

export interface VideoProgressBarProps {
  showControls: boolean;
  bufferedProgress: number;
  progress: number;
  duration: number;
  currentTime: number;
  events: EventSpan[];
  onScrubStart: () => void;
  onScrubEnd: () => void;
  onScrub: (progressPercent: number) => void;
  onHoverStart: () => void;
  onHoverEnd: () => void;
}

export interface VideoControlsProps {
  showControls: boolean;
  isPlaying: boolean;
  playbackRate: number;
  hasEvents: boolean;
  autoPlayMode: AutoPlayMode;
  hasNext: boolean;
  shuffleRemaining: number;
  onPressStart: () => void;
  onPressEnd: () => void;
  onHoverStart: () => void;
  onHoverEnd: () => void;
  onSkip: (seconds: number) => void;
  onPrevEvent: () => void;
  onNextEvent: () => void;
  onTogglePlay: () => void;
  onCycleAutoPlayMode: () => void;
  onChangeSpeed: (rate: number) => void;
  onOpenRename: () => void;
  onToggleDisqualified: () => void;
  onRotate: () => void;
  onClose: () => void;
  /** Current playback quality (only shown when `onChangeQuality` is provided). */
  quality?: VideoQuality;
  onChangeQuality?: (quality: VideoQuality) => void;
  /** Enter the highlight editor (only shown when provided). */
  onEditEvents?: () => void;
  /** True while the highlight editor is open, for the active button state. */
  isEditingEvents?: boolean;
}

/** Which edge of a selected event a nudge/edit targets. */
export type EventEdge = "start" | "end";

export interface EventEditorBarProps {
  events: EventSpan[];
  duration: number;
  currentTime: number;
  selectedIndex: number | null;
  onSelect: (index: number | null) => void;
  onSeek: (seconds: number) => void;
  /** Snapshot the draft before a drag so it becomes one undo step. */
  onBeginChange: () => void;
  onPreviewChange: (index: number, span: EventSpan) => void;
  onEndChange: () => void;
}

export interface EventEditToolbarProps {
  count: number;
  isSaving: boolean;
  canUndo: boolean;
  canRedo: boolean;
  /** False when the playhead sits on the shared edge of two events. */
  canAdd: boolean;
  /** Current pending rotation in degrees (0 when unrotated). */
  rotation: number;
  /** True when a rename is staged for the terminal Done. */
  isRenamed: boolean;
  /** True when events changed or a rename is staged (Done will commit). */
  hasChanges: boolean;
  /** True when the event draft differs from the loaded events. */
  eventsDirty: boolean;
  onAdd: () => void;
  onUndo: () => void;
  onRedo: () => void;
  onRevert: () => void;
  onRotate: () => void;
  onOpenRename: () => void;
  onCancel: () => void;
  onSave: () => void;
}

export interface EventInspectorProps {
  span: EventSpan | null;
  currentTime: number;
  onNudge: (edge: EventEdge, delta: number) => void;
  onSetToPlayhead: (edge: EventEdge) => void;
  onDelete: () => void;
}

export interface ConfirmDialogProps {
  title: string;
  description?: string;
  confirmLabel: string;
  /** Tint the confirm button red for a destructive action. */
  destructive?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export interface VideoRenameDialogProps {
  tempName: string;
  onChangeName: (name: string) => void;
  onDefault: () => void;
  onCancel: () => void;
  onSave: () => void;
}
