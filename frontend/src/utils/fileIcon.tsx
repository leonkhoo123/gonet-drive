import { File, FileImage, FileVideo, FileAudio, FileText, Terminal, FileCode } from "lucide-react";
import type { FileInterface } from "@/api/api-file";

export const getFileIcon = (file: FileInterface) => {
  if (file.media_type === "photo") {
    return <FileImage className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-primary" />;
  }
  if (file.media_type === "video") {
    return <FileVideo className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-purple-400" />;
  }
  if (file.media_type === "music") {
    return <FileAudio className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-pink-400" />;
  }
  if (file.media_type === "pdf") {
    return <FileText className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-red-500" />;
  }
  if (file.media_type === "text_documents") {
    const lowerName = file.name.toLowerCase();
    if (lowerName.endsWith('.yaml') || lowerName.endsWith('.yml')) {
      return <FileCode className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-yellow-500" />;
    }
    if (lowerName.endsWith('.sh') || lowerName.endsWith('.bash')) {
      return <Terminal className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-green-500" />;
    }
    return <FileText className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-gray-500" />;
  }
  return <File className="h-10 w-10 md:h-8 md:w-8 shrink-0 text-gray-400" />;
};
