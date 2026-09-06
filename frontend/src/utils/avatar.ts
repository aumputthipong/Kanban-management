/**
 * Placeholder avatar and group-dot colours, kept clear of priority semantics so pure
 * red stays available for "overdue".
 */
export const AVATAR_COLORS = [
  "bg-blue-500",
  "bg-violet-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-rose-500",
  "bg-cyan-500",
];

/**
 * Deterministic colour for an id — same id, same colour across views. First-char
 * modulo, not a hash: ids starting with the same letter collide.
 */
export function getAvatarColor(userId: string): string {
  return AVATAR_COLORS[userId.charCodeAt(0) % AVATAR_COLORS.length];
}
