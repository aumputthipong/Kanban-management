---
version: alpha
name: Turtask
description: Calm, focused design system for a multi-board Kanban ERP — neutral surfaces, a single accent for action, semantic colors for status chips.
colors:
  primary: "#1E40AF"
  secondary: "#475569"
  surface: "#FFFFFF"
  background: "#F8FAFC"
  on-primary: "#FFFFFF"
  on-surface: "#0F172A"
  success: "#047857"
  danger: "#B91C1C"
  surface-tint: "#EEF2FF"
  surface-elevated: "#FFFFFF"
  state-progress-bg: "#DBEAFE"
  state-progress-fg: "#1D4ED8"
  state-done-bg: "#D1FAE5"
  state-done-fg: "#047857"
  state-overdue-bg: "#FEE2E2"
  state-overdue-fg: "#B91C1C"
priority:
  high: "#DC2626"
  medium: "#F59E0B"
  low: "#10B981"
  none: "#94A3B8"
typography:
  h1:
    fontFamily: Inter
    fontSize: 2rem
    fontWeight: 700
    lineHeight: 2.5rem
    letterSpacing: -0.02em
  h2:
    fontFamily: Inter
    fontSize: 1.5rem
    fontWeight: 600
    lineHeight: 2rem
    letterSpacing: -0.01em
  body-md:
    fontFamily: Inter
    fontSize: 1rem
    fontWeight: 400
    lineHeight: 1.5rem
  label-sm:
    fontFamily: Inter
    fontSize: 0.75rem
    fontWeight: 500
    lineHeight: 1rem
    letterSpacing: 0.02em
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
rounded:
  sm: 2px
  md: 6px
  lg: 8px
  xl: 12px
  2xl: 16px
  full: 9999px
size:
  pill-h: 24px
  priority-bar-w: 3px
  tag-dot: 5px
  avatar-sm: 18px
  status-icon: 14px
  popover-max: 320px
breakpoints:
  md: 768px
  lg: 1024px
  xl: 1280px
  min-supported: 768px
components:
  board:
    backgroundColor: "{colors.background}"
    textColor: "{colors.on-surface}"
    padding: 24px
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.lg}"
    padding: 16px
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
    padding: 12px
  button-secondary:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
    padding: 12px
  tag-success:
    backgroundColor: "{colors.success}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.sm}"
    padding: 4px
  tag-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.sm}"
    padding: 4px
---

## Overview

Turtask is a real-time Kanban workspace. The UI must read as **calm and focused** — long sessions of triage, drag-and-drop, and review. The system favors neutral surfaces, a single restrained accent for action (deep indigo), and semantic colors reserved for status chips. Density is moderate: cards are scannable at a glance but never cramped.

The visual language pairs with the product's functional posture: type-safe end-to-end, no surprises. The design system mirrors that — a small, opinionated palette and a single sans-serif family across the whole product.

## Colors

The palette is built around a near-white surface, a deep-ink text color, and a single indigo accent that drives all primary interaction. Status colors (success, danger) are used **only** on chips, badges, and toasts — never on prose or layout.

- **Primary (#1E40AF):** Deep indigo for the primary call-to-action — "Create board", "Save card", confirm buttons. Used sparingly so it remains visually loud.
- **Secondary (#475569):** Slate for secondary actions, muted icons, and metadata captions.
- **Surface (#FFFFFF):** Card and panel background. The canvas where work lives.
- **Background (#F8FAFC):** Board canvas behind columns — slightly cooler than pure white so card edges read crisply.
- **On Primary (#FFFFFF):** Text and icon color placed on top of any saturated brand color.
- **On Surface (#0F172A):** Primary text on light surfaces. High-contrast ink for headings and body.
- **Success (#047857):** Done / completed status chips. Never used for prose.
- **Danger (#B91C1C):** Destructive confirmations, overdue badges, error toasts.

## Typography

Inter at four steps. One family, four weights, predictable rhythm.

- **h1 — 2rem / 700:** Page titles ("Board: Q2 Roadmap", "My Tasks"). One per view.
- **h2 — 1.5rem / 600:** Section headers, modal titles, column titles.
- **body-md — 1rem / 400:** Card titles, descriptions, form inputs.
- **label-sm — 0.75rem / 500 / +0.02em:** Metadata, chip labels, helper text. Slight letter-spacing keeps small caps legible at low weight.

Negative letter-spacing on h1 / h2 (`-0.02em`, `-0.01em`) tightens display type without making it feel cramped.

## Layout

Layout uses a 4-pixel base scale. Cards sit on the board canvas with `md` padding; modals and panels use `lg`. Vertical rhythm between stacked elements defaults to `sm` inside a card, `md` between cards.

- **`{spacing.xs}` (4px):** Chip inner padding, icon offsets.
- **`{spacing.sm}` (8px):** Tight stacks — card title to metadata, label to input.
- **`{spacing.md}` (16px):** Card content padding, gap between cards in a column.
- **`{spacing.lg}` (24px):** Board canvas padding, modal padding, gap between columns.
- **`{spacing.xl}` (32px):** Top-level page padding, section separation.

## Shapes

The radius scale is named to match Tailwind's steps, so a `rounded-*` class in
code maps 1:1 to a token here. `lg` (8px) is the workhorse container radius;
larger surfaces step up, chips step down. Larger surface ⇒ larger radius.

- **`{rounded.sm}` (2px):** Chips, tags, tiny toggles.
- **`{rounded.md}` (6px):** Inputs, compact buttons, small menu items.
- **`{rounded.lg}` (8px):** Cards, dropdown panels, most controls — the default container radius.
- **`{rounded.xl}` (12px):** Large cards, board sub-page panels, the calendar.
- **`{rounded.2xl}` (16px):** Modals and hero panels — the largest surfaces.
- **`{rounded.full}`:** Pills, avatars, status dots, filter chips.

> **Honesty note:** this system started life with a "two radii" ideal; in
> practice it settled into the five-step scale above. `sm` (2px) and `md` (6px)
> sit close together and are a candidate for future consolidation, but both are
> in active use today. Pick the step that matches the surface size — and don't
> reach for arbitrary `rounded-[Npx]` values; extend this named scale instead.

## Components

- **board:** Full-bleed canvas, `background` surface with `on-surface` ink, `lg` padding. Hosts the column list horizontally.
- **card:** White `surface` with `on-surface` ink, `lg` rounded, `md` padding. The atomic unit of work — title, optional description, assignee, due date, tags.
- **button-primary:** `primary` indigo with `on-primary` text, `md` rounded, `md` padding. Used once per view max for the dominant action.
- **button-secondary:** `secondary` slate with `on-primary` text, `md` rounded, `md` padding. For "Cancel", "Back", non-destructive secondary flows.
- **tag-success:** `success` green chip with `on-primary` text. Marks "Done" status, completed milestones.
- **tag-danger:** `danger` red chip with `on-primary` text. Marks overdue cards, blocked items, destructive confirmations.
- **pill-task:** Calendar task pill — `pill-h` (24px) tall, `priority-bar-w` (3px) vertical bar on the left in `priority.*`, `status-icon` on the inside, title in `body-md`, optional duration (`secondary` slate), `tag-dot` (5px), `avatar-sm` (18px). Background by state: todo = `surface`, in-progress = `state-progress-bg` + 2px progress bar in `state-progress-fg` at bottom, done = `state-done-bg` with check icon in `state-done-fg` (never strikethrough), overdue = `state-overdue-bg` with bold title and red duration. Hover → `surface-tint`.
- **span-bar:** Multi-day calendar task. `primary` indigo background, `on-primary` text, avatar at the head. Only the head/tail of the span carry `rounded.sm`; intermediate days are square so the bar reads as continuous.
- **chip-filter:** Calendar filter row. Outline default (`secondary` border, `on-surface` text); active state = `surface-tint` background + `primary` text. `rounded.full`, `label-sm` typography.
- **popover-card:** Hover preview / day-detail popover. `surface-elevated` with shadow-md, `rounded.md`, `lg` padding, `popover-max` (320px) wide. Contains **at most one** `button-primary` ("Open card"); secondary actions are text-only links in `secondary` slate.
- **today-cell:** Today's date cell. `surface-tint` background wash + a `primary` `rounded.full` disc around the date number.

## Responsive

Turtask รองรับ **viewport ≥ 768px** (tablet ขึ้นไป). ต่ำกว่านี้แสดงข้อความขอให้ใช้บนหน้าจอใหญ่กว่า — ไม่ใช่ design goal. Breakpoints align กับ Tailwind defaults: `md` 768px, `lg` 1024px, `xl` 1280px.

**Tier rules — ใช้ tier ที่*ใกล้เคียงที่สุด*แล้ว customize เฉพาะจุด:**

- **md (768–1023, tablet):** Workspace sidebar **collapsed by default** (`w-13`, icon-only) — toggle ปรากฏแต่ไม่ขยายค้าง. Board page padding `px-4`. Calendar density ลด (เอา hours label ออกจาก pill). Charts grid 1-column. Stat strips 1-column ถ้า card มี progress bar; ถ้า icon + count ล้วน ๆ 3-column ได้.
- **lg (1024–1279, laptop):** Sidebar expand ได้, user toggle ตามใจ. Board padding `px-6`. Charts grid 2-column. Stat strips 3-column.
- **xl (≥1280, desktop):** Padding `px-8`. Charts ถ้า ≥3 widget → 3-column. ที่เหลือคงเดิม.

**Component-scoped rules:**

- **Kanban board** (`tasks/page.tsx`): **คง horizontal scroll ทุก viewport** — column width fixed, ผู้ใช้ scroll/swipe ดู. ไม่ stack เป็น single-column บน tablet.
- **Calendar month grid:** คง 7 column ทุก viewport (สัปดาห์คือสัปดาห์). ที่ md ลด `pill` density ภายใน cell + ซ่อน duration text. Filter row → horizontal scroll ถ้าล้น.
- **Planning session view:** SessionSidebar collapsible (มี toggle), default collapsed ที่ md, expanded ที่ ≥lg. ที่ <lg และ sidebar expand → overlay (ไม่ push capture area).
- **BoardHeader tabs:** ที่ md ใช้ `overflow-x-auto` ให้ tabs scroll ได้แทนการ wrap. Label คงไว้ — ห้ามเหลือแต่ icon (loss of clarity).
- **Modals:** ใช้ `max-w-[90vw]` หรือต่ำกว่า — ห้าม fixed width เกิน 768px.

**Don'ts:**

- ❌ ห้ามใช้ `min-w-[XXXpx]` กับ container ที่อยู่ใน sidebar shell — sidebar กิน 56–224px อยู่แล้ว
- ❌ ห้ามตั้งสมมติฐานว่า viewport กว้าง ≥1280px (laptop หลายตัวคือ 1280×800 หลัง sidebar เหลือ <1056)
- ❌ ห้ามใช้ Tailwind `sm:` (640px) เพื่อเปลี่ยน layout — project นี้ skip `sm` tier (ไม่ support mobile). `sm:` ใช้ได้แต่เพื่อ fine-tune ภายใน tier `md` เท่านั้น
- ❌ ห้ามซ่อน feature ที่ critical (เช่น "Settings" tab, "Promote" button) ด้วย `hidden md:flex` — feature ทุกอย่างต้องเข้าถึงได้ทุก viewport ที่ support

## Do's and Don'ts

**Do**

- Use **one** `button-primary` per view. If two actions compete for primary, demote one to `button-secondary`.
- Reserve `success` / `danger` for status signals on chips and toasts.
- Stick to the four typography steps. New sizes mean a spec change, not an inline override.
- Let `background` and `surface` carry hierarchy — cards lift off the board canvas through the surface delta alone, not borders.

**Don't**

- Don't use `primary` indigo for prose, links inside cards, or icon defaults — it loses its meaning as the "act now" color.
- Don't tint `danger` red into prose or pink alerts. Status colors are component-scoped.
- Don't reach for an arbitrary `rounded-[Npx]` value — use a named step from the scale. Adding a *new* step is a spec change, not an inline override.
- Don't pair saturated colors against each other (e.g. `primary` text on `danger` background) — every paired surface in this system uses `on-primary` (white) for text on saturation.
- Don't use `priority.*` colors as a background — they live **only** on the 3px left bar of `pill-task` (and as a small disc inside the priority chip in a popover). Priority is signal, not fill.
- Don't use strikethrough to indicate "done" — that pattern leaked from the old calendar where both completed and past-due cards were struck out, making the two indistinguishable. Use the check icon + `state-done-bg` instead.
- Don't grow calendar cells to fit content. Every day cell is the same height; overflow becomes a "+N more" affordance that opens a `popover-card`.
- Don't reach for raw Tailwind color utilities (`bg-rose-50`, `bg-amber-100`, `text-emerald-600`) in calendar code. If a state needs a color, it goes in `state-*` / `priority.*` tokens first.
