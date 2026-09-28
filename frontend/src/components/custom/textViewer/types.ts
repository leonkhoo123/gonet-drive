import type { FileInterface } from "@/api/api-file";

export interface TextViewerModalProps {
  file: FileInterface | null;
  isOpen: boolean;
  onClose: () => void;
  /** Enables the edit/save flow. Personal files only; share viewers leave this off. */
  editable?: boolean;
  /** Called after a successful save so the parent can refresh its listing. */
  onSaved?: () => void;
}

export interface Baseline {
  size: number;
  modified: string;
}

export interface ConfirmState {
  title: string;
  description: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
}
