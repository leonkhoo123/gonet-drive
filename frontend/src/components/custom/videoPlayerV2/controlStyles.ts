/**
 * Shared control styling for the V2 video player.
 *
 * The player sits on black video, so a dark fill would make the neutral
 * controls vanish. Instead every control gets a very faint white wash (`/5`)
 * that hints at the button shape while leaving the white icon/label as the
 * thing you actually see. Tinted actions add their colour on top of the same
 * subtle wash.
 *
 * Controls are applied on top of the shadcn `ghost` Button, whose
 * `hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50` we
 * override here (tailwind-merge keeps the last conflicting utility, and the
 * explicit `dark:hover:*` pair cancels the variant's dark-theme hover).
 */
const CONTROL_BASE =
  "text-white hover:text-white dark:text-white dark:hover:text-white";

/** Neutral control: a barely-there white wash so it never fully disappears. */
export const CONTROL = `${CONTROL_BASE} bg-white/5 hover:bg-white/10 dark:hover:bg-white/10`;
/** Same faint wash, tinted for the rename action. */
export const CONTROL_GREEN = `${CONTROL_BASE} bg-emerald-500/30 hover:bg-emerald-500/45 dark:hover:bg-emerald-500/45`;
/** Same faint wash, tinted for the disqualified action. */
export const CONTROL_RED = `${CONTROL_BASE} bg-red-500/30 hover:bg-red-500/45 dark:hover:bg-red-500/45`;
/** Same faint wash, tinted for the rotate action. */
export const CONTROL_YELLOW = `${CONTROL_BASE} bg-yellow-400/30 hover:bg-yellow-400/45 dark:hover:bg-yellow-400/45`;
/** Same faint wash, tinted for shuffle mode. */
export const CONTROL_BLUE = `${CONTROL_BASE} bg-blue-500/30 hover:bg-blue-500/45 dark:hover:bg-blue-500/45`;

/** Panel surface for the speed/quality flyouts, dialogs and the inspector. */
export const PANEL = "border border-white/15 bg-black/85 text-white shadow-lg";

/** Selected state for the speed/quality step buttons, layered on the panel. */
export const CONTROL_ACTIVE = `${CONTROL_BASE} bg-white/25 font-semibold hover:bg-white/35 dark:hover:bg-white/35`;
