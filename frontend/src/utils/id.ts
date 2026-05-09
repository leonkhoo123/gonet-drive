export const generateOpId = (): string => {
    return window.crypto.randomUUID();
};
