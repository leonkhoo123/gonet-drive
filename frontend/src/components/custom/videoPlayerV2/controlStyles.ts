/**
 * Shared control styling for the V2 video player.
 *
 * The player always sits on black video, and the edit timeline darkens the
 * footage with `bg-black/60`. Every control mirrors that "darken" effect — a
 * translucent black fill with white text — so the transport column, the edit
 * column, flyouts and dialogs read as one chrome.
 *
 * Controls are applied on top of the shadcn `ghost` Button, whose
 * `hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50` we
 * override here (tailwind-merge keeps the last conflicting utility, and the
 * explicit `dark:hover:*` pair cancels the variant's dark-theme hover).
 */
const CONTROL_BASE =
  "text-white hover:text-white dark:text-white dark:hover:text-white";

/** Neutral control: the same darkened fill as the edit timeline (`bg-black/60`). */
export const CONTROL = `${CONTROL_BASE} bg-black/50 hover:bg-black/70 dark:hover:bg-black/70`;
/** Same darkened base, tinted for the rename action. */
export const CONTROL_GREEN = `${CONTROL_BASE} bg-emerald-500/40 hover:bg-emerald-500/60 dark:hover:bg-emerald-500/60`;
/** Same darkened base, tinted for the disqualified action. */
export const CONTROL_RED = `${CONTROL_BASE} bg-red-500/40 hover:bg-red-500/60 dark:hover:bg-red-500/60`;
/** Same darkened base, tinted for the rotate action. */
export const CONTROL_YELLOW = `${CONTROL_BASE} bg-yellow-400/40 hover:bg-yellow-400/60 dark:hover:bg-yellow-400/60`;
/** Same darkened base, tinted for shuffle mode. */
export const CONTROL_BLUE = `${CONTROL_BASE} bg-blue-500/40 hover:bg-blue-500/60 dark:hover:bg-blue-500/60`;

/** Panel surface for the speed/quality flyouts, dialogs and the inspector. */
export const PANEL = "border border-white/15 bg-black/85 text-white shadow-lg";

/** Selected state for the speed/quality step buttons, layered on the panel. */
export const CONTROL_ACTIVE = `${CONTROL_BASE} bg-white/25 font-semibold hover:bg-white/35 dark:hover:bg-white/35`;
