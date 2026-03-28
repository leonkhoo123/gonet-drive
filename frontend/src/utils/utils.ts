// src/lib/utils.ts

/**
 * Formats a raw number of bytes into a human-readable string (e.g., 1.2 MB).
 * * @param bytes The size of the file in bytes (number).
 * @param decimals The number of decimal places to use. Default is 1.
 * @returns A formatted string with the appropriate unit (B, KB, MB, GB, TB).
 */
export function formatBytes(bytes: number, decimals = 1): string {
    if (bytes === 0) return '0 Bytes';

    // 1024 is the standard base for binary units (KiB, MiB, GiB)
    const k = 1024; 
    
    // Units Array: B, KB, MB, GB, TB, PB, etc.
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    // Ensure the number of decimals is positive
    const dm = decimals < 0 ? 0 : decimals;

    // Use logarithm to find the correct index in the 'sizes' array
    // Math.floor(Math.log(bytes) / Math.log(k)) gives the exponent (0 for B, 1 for KB, 2 for MB, etc.)
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    // Calculate the final value and append the unit
    return String(parseFloat((bytes / Math.pow(k, i)).toFixed(dm))) + ' ' + sizes[i];
}

// Example usage:
// formatBytes(1590000000) -> "1.48 GB"
// formatBytes(54600000)   -> "52.1 MB"
// formatBytes(987)        -> "987 Bytes"

// src/lib/utils.ts (or wherever you keep utilities)

/**
 * Parses an ISO 8601 date string (e.g., 2025-08-14T...) and formats it
 * into a user-friendly, localized date and time string.
 * @param isoString The ISO 8601 date string.
 * @returns A formatted string (e.g., "Aug 14, 2025, 11:38 PM").
 */
export function formatLastModified(isoString: string): string {
    // 1. Create a Date object from the ISO string
    const date = new Date(isoString);

    // 2. Define formatting options
    const options: Intl.DateTimeFormatOptions = {
        year: 'numeric',
        month: 'short', // e.g., 'Aug'
        day: 'numeric', // e.g., '14'
        hour: 'numeric',
        minute: '2-digit',
        hour12: true, // Use AM/PM
        // The time zone will default to the user's local system time zone,
        // which is generally what you want for file modification times.
    };

    // 3. Use toLocaleDateString for localized formatting
    // 'en-US' is a good default, but you might use a dynamic locale from your app config
    return date.toLocaleDateString('en-US', options);
}

// Example Output for '2025-08-14T23:38:23.8941697+08:00' (in a typical US locale):
// "Aug 14, 2025, 11:38 PM"

export const encodePathToUrl = (path: string) => {
    if (path === '/.cloud_delete') return '/recycle_bin';
    if (path.startsWith('/.cloud_delete/')) return path.replace('/.cloud_delete/', '/recycle_bin/');
    return path;
};

export const decodeUrlToPath = (urlPath: string) => {
    if (urlPath === '/recycle_bin') return '/.cloud_delete';
    if (urlPath.startsWith('/recycle_bin/')) return urlPath.replace('/recycle_bin/', '/.cloud_delete/');
    return urlPath;
};