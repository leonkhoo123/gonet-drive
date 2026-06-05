import { createContext, use, useEffect, useRef, useState, type ReactNode } from 'react';
import { wsClient, type OperationMessage } from '../api/wsClient';

interface OperationProgressContextType {
    operations: Record<string, OperationMessage>;
    clearCompleted: () => void;
    dismissOperation: (opId: string) => void;
    addOrUpdateOperation: (msg: OperationMessage) => void;
}

const OperationProgressContext = createContext<OperationProgressContextType | undefined>(undefined);

const RESYNC_TIMEOUT_MS = 5000;

export function OperationProgressProvider({ children }: { children: ReactNode }) {
    const [operations, setOperations] = useState<Record<string, OperationMessage>>({});
    const operationsRef = useRef<Record<string, OperationMessage>>({});
    const resyncTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    operationsRef.current = operations;

    useEffect(() => {
        const unsubscribe = wsClient.subscribe((msg: OperationMessage) => {
            if (msg.opId) {
                setOperations(prev => ({
                    ...prev,
                    [msg.opId]: msg
                }));
            }
        });

        return () => {
            unsubscribe();
        };
    }, []);

    useEffect(() => {
        const handleReconnected = () => {
            const ops = operationsRef.current;
            let pendingCount = 0;

            for (const opId in ops) {
                const op = ops[opId];
                if (op.opStatus === 'queued' || op.opStatus === 'starting' || op.opStatus === 'in-progress') {
                    pendingCount++;
                    wsClient.send({ type: 'check_progress', opId });
                }
            }

            if (pendingCount === 0) return;

            if (resyncTimerRef.current) {
                clearTimeout(resyncTimerRef.current);
            }

            resyncTimerRef.current = setTimeout(() => {
                resyncTimerRef.current = null;
                setOperations(prev => {
                    const result: Record<string, OperationMessage> = {};
                    for (const key in prev) {
                        const op = prev[key];
                        if (op.opStatus !== 'queued' && op.opStatus !== 'starting' && op.opStatus !== 'in-progress') {
                            result[key] = op;
                        }
                    }
                    return result;
                });
            }, RESYNC_TIMEOUT_MS);
        };

        window.addEventListener('ws-reconnected', handleReconnected);
        return () => {
            window.removeEventListener('ws-reconnected', handleReconnected);
            if (resyncTimerRef.current) {
                clearTimeout(resyncTimerRef.current);
            }
        };
    }, []);

    const clearCompleted = () => {
        setOperations(prev => {
            const result: Record<string, OperationMessage> = {};
            for (const key in prev) {
                const op = prev[key];
                if (op.opStatus !== 'completed' && op.opStatus !== 'error' && op.opStatus !== 'aborted') {
                    result[key] = op;
                }
            }
            return result;
        });
    };

    const dismissOperation = (opId: string) => {
        setOperations(prev => {
            return Object.fromEntries(
                Object.entries(prev).filter(([key]) => key !== opId)
            ) as Record<string, OperationMessage>;
        });
    };

    const addOrUpdateOperation = (msg: OperationMessage) => {
        if (msg.opId) {
            setOperations(prev => ({
                ...prev,
                [msg.opId]: msg
            }));
        }
    };

    return (
        <OperationProgressContext value={{ operations, clearCompleted, dismissOperation, addOrUpdateOperation }}>
            {children}
        </OperationProgressContext>
    );
}

export function useOperationProgress() {
    const context = use(OperationProgressContext);
    if (!context) {
        throw new Error('useOperationProgress must be used within an OperationProgressProvider');
    }
    return context;
}
