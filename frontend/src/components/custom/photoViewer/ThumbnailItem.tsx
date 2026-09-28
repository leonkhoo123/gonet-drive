import React, { useEffect, useRef } from "react";
import { type FileInterface } from "@/api/api-file";
import { thumbUrl } from "./thumbUrl";

export const ThumbnailItem = React.memo(
  ({
    file,
    isActive,
    index,
    onGoTo,
    onLoad,
  }: {
    file: FileInterface;
    isActive: boolean;
    index: number;
    onGoTo: (idx: number) => void;
    onLoad: (index: number) => void;
  }) => {
    const imgRef = useRef<HTMLImageElement>(null);

    useEffect(() => {
      const img = imgRef.current;
      if (img && img.complete && img.naturalWidth > 0) {
        onLoad(index);
      }
    }, [index, onLoad]);

    return (
      <button
        onClick={(e) => {
          e.stopPropagation();
          onGoTo(index);
        }}
        className={`flex-shrink-0 h-16 transition-all focus:outline-none ${
          isActive
            ? "scale-105"
            : "opacity-60 hover:opacity-100"
        }`}
      >
        <img
          ref={imgRef}
          src={thumbUrl(file)}
          alt={file.name}
          className={`h-full w-auto rounded-md ${
            isActive
              ? "ring-2 ring-white ring-offset-1 ring-offset-transparent"
              : ""
          }`}
          loading="lazy"
          decoding="async"
          onLoad={() => { onLoad(index); }}
        />
      </button>
    );
  }
);

ThumbnailItem.displayName = "ThumbnailItem";
