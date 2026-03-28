import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
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
            const next = { ...prev };
            for (const key in next) {
                if (next[key].opStatus === 'completed' || next[key].opStatus === 'error' || next[key].opStatus === 'aborted') {
                    // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
                    delete next[key];
                }
            }
            return next;
        });
    };

    const dismissOperation = (opId: string) => {
        setOperations(prev => {
            const next = { ...prev };
            // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
            delete next[opId];
            return next;
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
        <OperationProgressContext.Provider value={{ operations, clearCompleted, dismissOperation, addOrUpdateOperation }}>
            {children}
        </OperationProgressContext.Provider>
    );
}

export function useOperationProgress() {
    const context = useContext(OperationProgressContext);
    if (!context) {
        throw new Error('useOperationProgress must be used within an OperationProgressProvider');
    }
    return context;
}
