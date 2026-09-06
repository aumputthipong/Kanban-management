/**
 * Date helpers for cards, activities and dashboards. All deltas are against
 * today-at-midnight local, so a card due today reads "Today" whatever the clock says.
 */

/** "Overdue by N days" — assumes the due date is in the past. */
export const getOverdueText = (dueDateStr: string) => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const dueDate = new Date(dueDateStr);
  dueDate.setHours(0, 0, 0, 0);

  const diffTime = today.getTime() - dueDate.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 1) return "Overdue by 1 day";
  return `Overdue by ${diffDays} days`;
};

/** Long Thai date — e.g. "1 เมษายน 2569". Used as the canonical "absolute" form. */
export const formatThaiDate = (dateStr: string) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString("th-TH", {
    year: "numeric",
    month: "long", // use "short" for an abbreviated month
    day: "numeric",
  });
};

/** "Due today" / "Due tomorrow" / "In N days" — assumes due date is in the future. */
export const getDaysRemainingText = (dueDateStr: string) => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const dueDate = new Date(dueDateStr);
  dueDate.setHours(0, 0, 0, 0);

  const diffTime = dueDate.getTime() - today.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 0) return "Due today";
  if (diffDays === 1) return "Due tomorrow";
  return `In ${diffDays} days`;
};

// YYYY-MM-DD key in local time — safe for grouping by calendar day.
export const dateKey = (d: Date): string => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};

// "Today" / "Tomorrow" / "Wed, Apr 30" — for sub-group headers.
export const formatDayLabel = (date: Date, today: Date): string => {
  const t = new Date(today);
  t.setHours(0, 0, 0, 0);
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  const diff = Math.round((d.getTime() - t.getTime()) / 86400000);
  if (diff === 0) return "Today";
  if (diff === 1) return "Tomorrow";
  return d.toLocaleDateString("en-US", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
};

// Compact relative label suitable for dense list rows (My Tasks).
// Falls back to formatThaiDate if more than 7 days out.
export const formatRelativeDueDate = (dueDateStr: string): string => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const dueDate = new Date(dueDateStr);
  dueDate.setHours(0, 0, 0, 0);

  const diffDays = Math.round(
    (dueDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24),
  );

  if (diffDays < 0) {
    const days = Math.abs(diffDays);
    return days === 1 ? "1 day overdue" : `${days} days overdue`;
  }
  if (diffDays === 0) return "Today";
  if (diffDays === 1) return "Tomorrow";
  if (diffDays <= 7) return `In ${diffDays} days`;
  return formatThaiDate(dueDateStr);
};
