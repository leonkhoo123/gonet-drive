import { useState, useEffect, useRef } from 'react';
import { useAutoAnimate } from '@formkit/auto-animate/react';
import { useOperationProgress } from '../../context/OperationProgressContext';
import { Progress } from '../ui/progress';
import { Button } from '../ui/button';
import { X, ChevronDown, CheckCircle2, AlertCircle, Loader2, Files, Trash2, Edit, Move, Copy, UploadCloud } from 'lucide-react';
import type { OperationMessage } from '@/api/wsClient';
import { cancelOperation, uploadControllers, cancelledUploads } from '@/api/api-file';

export function OperationQueueProgress() {
    const { operations, clearCompleted, dismissOperation } = useOperationProgress();
    const [isExpanded, setIsExpanded] = useState(false);
    const panelRef = useRef<HTMLDivElement>(null);
    const prevOpIdsRef = useRef<Set<string>>(new Set(Object.keys(operations)));
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const [listRef] = useAutoAnimate<HTMLDivElement>();

    useEffect(() => {
        if (isExpanded && scrollContainerRef.current) {
            scrollContainerRef.current.scrollTop = 0;
        }
    }, [operations, isExpanded]);

    useEffect(() => {
        const currentIds = Object.keys(operations);
        const prevIds = prevOpIdsRef.current;
        
        let hasNewTargetOp = false;
        for (const id of currentIds) {
            if (!prevIds.has(id)) {
                const opType = operations[id].opType;
                if (['copy', 'move', 'delete', 'delete_permanent', 'upload'].includes(opType)) {
                    hasNewTargetOp = true;
                    break;
                }
            }
        }

        if (hasNewTargetOp) {
            setIsExpanded(true);
        }
        
        prevOpIdsRef.current = new Set(currentIds);
    }, [operations]);

    useEffect(() => {
        function handleClickOutside(event: MouseEvent | TouchEvent) {
            if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
                setIsExpanded(false);
            }
        }

        if (isExpanded) {
            document.addEventListener('mousedown', handleClickOutside);
            document.addEventListener('touchstart', handleClickOutside);
        }

        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
            document.removeEventListener('touchstart', handleClickOutside);
        };
    }, [isExpanded]);

    const opsList = Object.values(operations).reverse().sort((a, b) => {
        const getStatusWeight = (status: string) => {
            switch (status) {
                case 'in-progress': return 3;
                case 'starting': return 2;
                case 'queued': return 1;
                default: return 0; // completed, error, etc
            }
        };

        const weightA = getStatusWeight(a.opStatus);
        const weightB = getStatusWeight(b.opStatus);
        
        if (weightA > weightB) return -1;
        if (weightA < weightB) return 1;

        return 0;
    });

    const activeOps = opsList.filter(op => op.opStatus === 'in-progress' || op.opStatus === 'starting' || op.opStatus === 'queued');
    const hasActiveOps = activeOps.length > 0;
    const hasCompletedOps = opsList.some(op => op.opStatus === 'completed' || op.opStatus === 'error' || op.opStatus === 'aborted');

    const title = hasActiveOps 
        ? `${String(activeOps.length)} operation${activeOps.length > 1 ? 's' : ''} in progress` 
        : (opsList.length > 0 ? `All operations completed` : `No operation`);

    const getHeaderIcon = () => {
        if (hasActiveOps) return <Loader2 className="w-4 h-4 animate-spin text-primary" />;
        if (opsList.length > 0) return <CheckCircle2 className="w-4 h-4 text-green-500" />;
        return <CheckCircle2 className="w-4 h-4 text-muted-foreground" />;
    };

    const getMobileIcon = () => {
        if (hasActiveOps) return <Loader2 className="w-6 h-6 animate-spin text-primary" />;
        if (opsList.length > 0) return <CheckCircle2 className="w-6 h-6 text-green-500" />;
        return <CheckCircle2 className="w-6 h-6 text-muted-foreground" />;
    };

    const getIconForType = (type: string) => {
        switch (type) {
            case 'copy': return <Copy className="w-4 h-4" />;
            case 'move': return <Move className="w-4 h-4" />;
            case 'delete': return <Trash2 className="w-4 h-4" />;
            case 'delete_permanent': return <Trash2 className="w-4 h-4 text-red-500" />;
            case 'rename': return <Edit className="w-4 h-4" />;
            case 'upload': return <UploadCloud className="w-4 h-4" />;
            default: return <Files className="w-4 h-4" />;
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'completed': return <CheckCircle2 className="w-5 h-5 text-green-500" />;
            case 'error': return <AlertCircle className="w-5 h-5 text-red-500" />;
            case 'aborted': return <AlertCircle className="w-5 h-5 text-orange-500" />;
            case 'in-progress':
            case 'starting':
            case 'queued': return <Loader2 className="w-5 h-5 text-primary animate-spin" />;
            default: return null;
        }
    };

    const renderOpItem = (op: OperationMessage) => (
        <div key={op.opId} className="flex flex-col border rounded-md p-2 bg-background shadow-sm">
            <div className="flex justify-between items-center">
                <div className="flex items-center gap-2 min-w-0 pr-2">
                    <div className="p-1.5 bg-muted rounded-md text-foreground shrink-0">
                        {getIconForType(op.opType)}
                    </div>
                    <div className="flex flex-col min-w-0 justify-center">
                        {op.opName ? (
                            <span className="text-sm text-left break-words line-clamp-2" title={op.opName}>
                                {op.opName}
                            </span>
                        ) : (
                            <span className="text-sm capitalize truncate text-left">{op.opType}</span>
                        )}
                        <span className="text-xs text-muted-foreground capitalize text-left">{op.opStatus}</span>
                    </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                    {getStatusIcon(op.opStatus)}
                    {(op.opStatus === 'completed' || op.opStatus === 'error' || op.opStatus === 'aborted') && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-5 w-5"
                            onClick={(e) => { e.stopPropagation(); dismissOperation(op.opId); }}
                        >
                            <X className="w-3 h-3" />
                        </Button>
                    )}
                    {(op.opStatus === 'in-progress' || op.opStatus === 'starting' || op.opStatus === 'queued') && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-5 w-5"
                            onClick={(e) => { 
                                e.stopPropagation(); 
                                void (async () => {
                                    try {
                                        if (op.opType === 'upload') {
                                            cancelledUploads.add(op.opId); // Mark as cancelled for queued uploads
                                            const controller = uploadControllers.get(op.opId);
                                            if (controller) {
                                                controller.abort();
                                            } else {
                                                await cancelOperation(op.opId);
                                            }
                                        } else {
                                            await cancelOperation(op.opId);
                                        }
                                    } catch (error) {
                                        console.error('Failed to cancel operation', error);
                                    }
                                })();
                            }}
                        >
                            <X className="w-3 h-3 text-muted-foreground hover:text-red-500" />
                        </Button>
                    )}
                </div>
            </div>

            {op.opStatus === 'in-progress' && (
                <div className="flex flex-col gap-1 mt-2">
                    {op.opPercentage !== undefined && op.opPercentage !== null && (
                        <Progress value={op.opPercentage} className="h-1.5" />
                    )}
                    <div className="flex justify-between text-[10px] text-muted-foreground mt-1">
                        <span>{op.opFileCount ? `Files: ${op.opFileCount}` : ''}</span>
                        <div className="flex gap-2">
                            <span>{op.opSpeed ?? ''}</span>
                            <span>{op.opPercentage !== undefined && op.opPercentage !== null ? `${String(Math.round(op.opPercentage))}%` : ''}</span>
                        </div>
                    </div>
                </div>
            )}
            
            {op.error && op.opStatus !== 'aborted' && (
                <div className="text-xs text-red-500 mt-1 bg-red-50 p-1.5 rounded border border-red-100">
                    {op.error}
                </div>
            )}
        </div>
    );

    const renderOpsListContent = () => {
        if (opsList.length === 0) {
            return (
                <div className="text-center text-sm text-muted-foreground py-4">
                    No recent operations
                </div>
            );
        }
        return (
            <div ref={listRef} className="flex flex-col gap-2">
                {opsList.map(renderOpItem)}
            </div>
        );
    };

    return (
        <div 
            ref={panelRef}
            className="absolute bottom-4 left-4 z-50 flex flex-col items-start justify-end"
        >
            {/* Pop up panel with animation */}
            <div 
                className={`
                    w-[calc(100vw-2rem)] md:w-96 max-w-md shadow-xl bg-background rounded-lg border flex flex-col overflow-hidden mb-2
                    transition-all duration-300 ease-in-out origin-bottom-left
                    ${isExpanded ? 'opacity-100 scale-100 translate-y-0' : 'opacity-0 scale-95 translate-y-4 pointer-events-none absolute bottom-12'}
                `}
            >
                <div 
                    className="bg-muted p-3 flex justify-between items-center cursor-pointer border-b"
                    onClick={() => { setIsExpanded(false); }}
                >
                    <div className="flex items-center gap-2 font-medium text-sm">
                        {getHeaderIcon()}
                        <span>{title}</span>
                    </div>
                    <div className="flex items-center gap-1">
                        {hasCompletedOps && (
                            <Button 
                                variant="ghost" 
                                size="sm" 
                                className="h-6 px-2 text-xs rounded-full" 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    clearCompleted();
                                }}
                                title="Clear completed tasks"
                            >
                                Clear
                            </Button>
                        )}
                        <Button variant="ghost" size="icon" className="h-6 w-6 rounded-full">
                            <ChevronDown className="w-4 h-4" />
                        </Button>
                    </div>
                </div>
                <div ref={scrollContainerRef} className="max-h-[30vh] overflow-y-auto p-2 bg-card">
                    {renderOpsListContent()}
                </div>
            </div>
            
            {/* Floating Action Button */}
            <div 
                className={`
                    w-12 h-12 bg-background shadow-lg rounded-full border flex items-center justify-center cursor-pointer hover:bg-muted transition-all duration-300
                    ${isExpanded ? 'bg-muted shadow-md' : ''}
                `}
                onClick={() => { setIsExpanded(!isExpanded); }}
                title="Operation Queue"
            >
                {getMobileIcon()}
            </div>
        </div>
    );
}
