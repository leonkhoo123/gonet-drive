/**
 * Traditional (non-glass) control styling for the V2 video player.
 *
 * Light-mode gray surfaces with dark text and no border, and deliberately no
 * backdrop blur. The player always sits on a black video, so the same look is
 * used in every theme; the `dark:hover:*` overrides cancel the ghost Button
 * variant's theme-driven hover styles.
 */
const CONTROL_BASE =
  "text-gray-900 hover:text-gray-900 dark:hover:text-gray-900";

/** Neutral gray button. */
export const CONTROL = `${CONTROL_BASE} bg-gray-200/85 hover:bg-gray-300/85 dark:hover:bg-gray-300/85`;
/** Same treatment, tinted for the rename action. */
export const CONTROL_GREEN = `${CONTROL_BASE} bg-green-400/85 hover:bg-green-400/95 dark:hover:bg-green-400/95`;
/** Same treatment, tinted for the disqualified action. */
export const CONTROL_RED = `${CONTROL_BASE} bg-red-400/85 hover:bg-red-400/95 dark:hover:bg-red-400/95`;
/** Same treatment, tinted for the rotate action. */
export const CONTROL_YELLOW = `${CONTROL_BASE} bg-yellow-400/85 hover:bg-yellow-400/95 dark:hover:bg-yellow-400/95`;
/** Same treatment, tinted for shuffle mode. */
export const CONTROL_BLUE = `${CONTROL_BASE} bg-blue-400/85 hover:bg-blue-400/95 dark:hover:bg-blue-400/95`;

/** Panel surface for the speed flyout and the rename dialog. */
export const PANEL = "bg-gray-100/95 text-gray-900 shadow-lg";

/** Selected state for the speed step buttons, layered on the panel. */
export const CONTROL_ACTIVE = `${CONTROL_BASE} bg-gray-400/85 font-semibold hover:bg-gray-400/95 dark:hover:bg-gray-400/95`;
