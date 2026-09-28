/**
 * Selectable playback speeds, ordered slow -> fast.
 *
 * Shared by the speed flyout and the `<` / `>` keyboard shortcuts so both step
 * through the same set of rates.
 */
export const PLAYBACK_SPEEDS = [0.25, 0.5, 0.75, 1, 1.25, 1.5, 2] as const;

export type PlaybackSpeed = (typeof PLAYBACK_SPEEDS)[number];

/** Tolerance used when matching a live `playbackRate` against the steps. */
const EPSILON = 1e-6;

/**
 * Step the playback rate one notch faster (+1) or slower (-1).
 *
 * Stays within `PLAYBACK_SPEEDS` and clamps at the fastest/slowest step.
 */
export function stepPlaybackRate(current: number, direction: 1 | -1): number {
  if (direction > 0) {
    return (
      PLAYBACK_SPEEDS.find((speed) => speed > current + EPSILON) ??
      PLAYBACK_SPEEDS[PLAYBACK_SPEEDS.length - 1]
    );
  }
  const slower = [...PLAYBACK_SPEEDS]
    .reverse()
    .find((speed) => speed < current - EPSILON);
  return slower ?? PLAYBACK_SPEEDS[0];
}
