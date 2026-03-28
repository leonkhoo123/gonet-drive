import React, { useRef, useState } from 'react';

interface TruncatedTextProps extends React.HTMLAttributes<HTMLSpanElement> {
  text: string;
}

export function TruncatedText({ text, className, ...props }: TruncatedTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [title, setTitle] = useState<string | undefined>(undefined);

  const handleMouseEnter = (e: React.MouseEvent<HTMLSpanElement>) => {
    if (textRef.current) {
      if (textRef.current.scrollWidth > textRef.current.clientWidth) {
        setTitle(text);
      } else {
        setTitle(undefined);
      }
    }
    props.onMouseEnter?.(e);
  };

  return (
    <span
      ref={textRef}
      className={`truncate ${className ?? ''}`}
      onMouseEnter={handleMouseEnter}
      title={title}
      {...props}
    >
      {text}
    </span>
  );
}
