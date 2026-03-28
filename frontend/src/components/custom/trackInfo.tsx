import { useEffect, useRef, useState } from "react";
import { Marquee } from "../ui/marquee";
import { formatTime } from "@/lib/utils";

const TrackInfo = ({ text, currentTime, duration, onClick }: { text: string, currentTime: number, duration: number, onClick?: () => void }) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const textRef = useRef<HTMLSpanElement>(null);
    const [isOverflowing, setIsOverflowing] = useState(false);

    useEffect(() => {
        const checkOverflow = () => {
            if (containerRef.current && textRef.current) {
                const availableWidth = containerRef.current.clientWidth;
                const textWidth = textRef.current.scrollWidth;
                setIsOverflowing(textWidth > availableWidth);
            }
        };

        checkOverflow();
        const resizeObserver = new ResizeObserver(checkOverflow);

        if (containerRef.current) {
            resizeObserver.observe(containerRef.current);
        }

        return () => { resizeObserver.disconnect(); };
    }, [text]);

    return (
        <div className={`flex items-center justify-center gap-0 flex-1 min-w-0 px-1 md:px-2 ${onClick ? 'cursor-pointer hover:opacity-80' : ''}`} onClick={onClick}>
            <div className="flex flex-col min-w-0 overflow-hidden items-center text-center w-full">
                <div ref={containerRef} className="w-full overflow-hidden flex items-center justify-center relative">
                    <span
                        ref={textRef}
                        className="absolute opacity-0 pointer-events-none whitespace-nowrap text-sm font-medium"
                        aria-hidden="true"
                    >
                        {text}
                    </span>
                    {isOverflowing ? (
                        <Marquee className="[--duration:7s] p-0 w-full" pauseOnHover>
                            <span className="text-sm font-medium px-4">{text}</span>
                        </Marquee>
                    ) : (
                        <span className="block text-sm font-medium truncate w-full" title={text}>
                            {text}
                        </span>
                    )}
                </div>
                <span className="block text-[10px] text-muted-foreground md:hidden truncate max-w-full mt-0.5">
                    {formatTime(currentTime)} / {formatTime(duration)}
                </span>
            </div>
        </div>
    );
};

export default TrackInfo;