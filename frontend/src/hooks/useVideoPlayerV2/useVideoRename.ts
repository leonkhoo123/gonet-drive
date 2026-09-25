import { useCallback, useState } from "react";

/**
 * Rename / disqualify / rotation-agnostic flag state for the video player.
 * Keeps the edited name, the rename dialog and the "disqualified" toggle in
 * one place and returns the handlers the controls and keyboard need.
 */
export function useVideoRename(fileName: string) {
  const [newName, setNewname] = useState("");
  const [isNewName, setisNewname] = useState(false);
  const [disqualified, setDisqualified] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [tempName, setTempName] = useState("");

  const handleRenameSave = useCallback(() => {
    let finalName = tempName.trim();
    const ext = fileName.includes(".")
      ? fileName.substring(fileName.lastIndexOf("."))
      : "";

    if (!finalName.includes(".") && ext) finalName += ext;

    if (finalName !== fileName) {
      setNewname(finalName);
      setisNewname(true);
    }

    setShowRenameModal(false);
  }, [tempName, fileName]);

  const handleRenameDefault = () => {
    setNewname("");
    setisNewname(false);
    setShowRenameModal(false);
  };

  const handleRenameCancel = useCallback(() => {
    setShowRenameModal(false);
  }, []);

  const openRenameModal = () => {
    setTempName(newName);
    setShowRenameModal(true);
  };

  const handleDisqualified = () => {
    setisNewname(false);
    setNewname("");
    setDisqualified(!disqualified);
  };

  return {
    newName,
    isNewName,
    disqualified,
    showRenameModal,
    tempName,
    setTempName,
    handleRenameSave,
    handleRenameDefault,
    handleRenameCancel,
    openRenameModal,
    handleDisqualified,
  };
}
