import { createContext, use, useEffect, useState, type ReactNode } from 'react';
import { wsClient, type OperationMessage } from '../api/wsClient';

interface OperationProgressContextType {
    operations: Record<string, OperationMessage>;
    clearCompleted: () => void;
    dismissOperation: (opId: string) => void;
    addOrUpdateOperation: (msg: OperationMessage) => void;
}

const OperationProgressContext = createContext<OperationProgressContextType | undefined>(undefined);

export function OperationProgressProvider({ children }: { children: ReactNode }) {
    const [operations, setOperations] = useState<Record<string, OperationMessage>>({});

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
