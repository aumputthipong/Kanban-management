// Icon keys mirror the backend `boards.icon` enum — keep in sync.
import { LayoutGrid, Rocket, Target, Zap, Bug, type LucideIcon } from "lucide-react";

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

export const BOARD_ICONS = {
  board: LayoutGrid,
  rocket: Rocket,
  target: Target,
  bolt: Zap,
  bug: Bug,
} satisfies Record<string, LucideIcon>;

export type BoardIconKey = keyof typeof BOARD_ICONS;

export function boardColor(color?: string | null): string {
  return color && color.trim() !== "" ? color : DEFAULT_BOARD_COLOR;
}

// Index BOARD_ICONS directly — react-hooks/static-components needs a stable reference.
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
