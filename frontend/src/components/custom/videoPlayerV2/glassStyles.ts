/**
 * Light-mode glassmorphism shared by the V2 video controls and rename dialog.
 *
 * The colors are hard-coded instead of theme tokens on purpose: the player
 * always sits on top of a black video, so it must keep the same frosted light
 * look regardless of the app theme. The `dark:hover:*` overrides cancel the
 * ghost Button variant's theme-driven hover styles, and the focus overrides
 * drop the themed border/ring so no border is ever drawn.
 */
const GLASS_BASE =
  "backdrop-blur-md text-black hover:text-black dark:hover:text-black focus-visible:border-transparent focus-visible:ring-black/30";

/** Neutral frosted button. */
export const GLASS = `${GLASS_BASE} bg-white/55 hover:bg-white/70 dark:hover:bg-white/70`;
/** Same glass treatment, tinted for the rename action. */
export const GLASS_GREEN = `${GLASS_BASE} bg-green-400/55 hover:bg-green-400/75 dark:hover:bg-green-400/75`;
/** Same glass treatment, tinted for the disqualified action. */
export const GLASS_RED = `${GLASS_BASE} bg-red-400/55 hover:bg-red-400/75 dark:hover:bg-red-400/75`;
/** Same glass treatment, tinted for the rotate action. */
export const GLASS_YELLOW = `${GLASS_BASE} bg-yellow-400/55 hover:bg-yellow-400/75 dark:hover:bg-yellow-400/75`;
/** Same glass treatment, tinted for shuffle mode. */
export const GLASS_BLUE = `${GLASS_BASE} bg-blue-400/55 hover:bg-blue-400/75 dark:hover:bg-blue-400/75`;

/** Frosted panel surface for the speed flyout and the rename dialog. */
export const GLASS_PANEL = "backdrop-blur-md bg-white/55 text-black shadow-lg";

/** Selected state for the speed step buttons, layered on the frosted panel. */
export const GLASS_ACTIVE = `${GLASS_BASE} bg-black/15 hover:bg-black/20 dark:hover:bg-black/20`;
