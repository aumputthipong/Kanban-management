// lib/boardAppearance.tsx
//
// Single source of truth for a board's visual identity (accent colour + glyph).
// The settings picker (GeneralSection), the project-list cards, the board
// Header (BoardHeader), and the workspace Sidebar all read from here so a board
// looks identical everywhere it appears — Header is "you are here", Sidebar is
// navigation, but both share the same glyph + accent.
//
// The colour palette is a fixed set of per-board *identity* colours — not
// semantic status colours. The default accent is the design-system primary
// (#1E40AF). The icon keys mirror the backend's `oneof` validator on
// boards.icon (board/rocket/target/bolt/bug) — keep the two in sync.
import { LayoutGrid, Rocket, Target, Zap, Bug, type LucideIcon } from "lucide-react";

/** Selectable accent colours for a board. First entry = default. */
export const BOARD_COLORS = [
  "#1E40AF", // primary (design token)
  "#0D9488", // teal
  "#7C3AED", // violet
  "#B45309", // amber-brown
  "#BE185D", // pink
  "#0F172A", // slate-ink
] as const;

export const DEFAULT_BOARD_COLOR = BOARD_COLORS[0];
export const DEFAULT_BOARD_ICON = "board";

/** Glyph key → lucide icon. Keys match the backend `boards.icon` enum. */
export const BOARD_ICONS = {
  board: LayoutGrid,
  rocket: Rocket,
  target: Target,
  bolt: Zap,
  bug: Bug,
} satisfies Record<string, LucideIcon>;

export type BoardIconKey = keyof typeof BOARD_ICONS;

/** Resolve a board's accent colour, falling back to the default. */
export function boardColor(color?: string | null): string {
  return color && color.trim() !== "" ? color : DEFAULT_BOARD_COLOR;
}

/**
 * Renders a board's glyph. Resolving the icon by indexing the constant
 * BOARD_ICONS map (rather than via a function call) keeps the
 * react-hooks/static-components rule happy — the rendered component is a stable
 * reference, not one "created during render".
 */
export function BoardGlyph({
  icon,
  size,
  className,
}: {
  icon?: string | null;
  size?: number;
  className?: string;
}) {
  const Icon =
    (icon && BOARD_ICONS[icon as BoardIconKey]) || BOARD_ICONS[DEFAULT_BOARD_ICON];
  return <Icon size={size} className={className} />;
}
