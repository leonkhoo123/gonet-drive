export const generateOpId = (): string => {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (window.crypto?.randomUUID) {
        return window.crypto.randomUUID();
    }
    
    // Fallback for environments where crypto.randomUUID is not available (e.g., non-HTTPS)
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
};
