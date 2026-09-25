import type { FileInterface } from "@/api/api-file";

/**
 * Session-only shuffle bookkeeping. A separate "played" set is kept per folder
 * so shuffle-bag mode plays each clip once before stopping. Module scope means
 * the bag survives the player being closed and reopened during a session, but
 * resets on page reload — matching the "per folder, per session" behaviour.
 */
const playedByFolder = new Map<string, Set<string>>();

/** Mark a clip as played in its folder's shuffle bag. */
export function markVideoPlayed(folderPath: string, filePath: string): void {
  let played = playedByFolder.get(folderPath);
  if (!played) {
    played = new Set<string>();
    playedByFolder.set(folderPath, played);
  }
  played.add(filePath);
}

/** Videos of a folder that shuffle has not played yet, in list order. */
export function getUnplayedVideos(
  folderPath: string,
  videoFiles: FileInterface[]
): FileInterface[] {
  const played = playedByFolder.get(folderPath);
  if (!played || played.size === 0) return [...videoFiles];
  return videoFiles.filter((file) => !played.has(file.path));
}

/** Forget a folder's shuffle bag so the next shuffle starts a fresh cycle. */
export function resetPlayedVideos(folderPath: string): void {
  playedByFolder.delete(folderPath);
}
