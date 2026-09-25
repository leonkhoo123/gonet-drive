import type { RefObject, TouchEvent } from "react";
import type { FileInterface } from "@/api/api-file";
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
}

export interface VideoControlsProps {
  showControls: boolean;
  isPlaying: boolean;
  playbackRate: number;
  hasEvents: boolean;
  onPressStart: () => void;
  onPressEnd: () => void;
  onSkip: (seconds: number) => void;
  onPrevEvent: () => void;
  onNextEvent: () => void;
  onTogglePlay: () => void;
  onChangeSpeed: (rate: number) => void;
  onOpenRename: () => void;
  onToggleDisqualified: () => void;
  onRotate: () => void;
  onClose: () => void;
}

export interface VideoRenameDialogProps {
  tempName: string;
  onChangeName: (name: string) => void;
  onDefault: () => void;
  onCancel: () => void;
  onSave: () => void;
}
