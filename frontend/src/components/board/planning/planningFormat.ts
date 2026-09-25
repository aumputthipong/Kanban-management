const REL_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["second", 60],
  ["minute", 60],
  ["hour", 24],
  ["day", 7],
  ["week", 4.34524],
  ["month", 12],
  ["year", Number.POSITIVE_INFINITY],
];

const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function formatRelativeFromNow(iso: string): string {
  const diffMs = new Date(iso).getTime() - Date.now();
  let diff = diffMs / 1000;
  for (const [unit, divisor] of REL_UNITS) {
    if (Math.abs(diff) < divisor) {
      return rtf.format(Math.round(diff), unit);
    }
    diff /= divisor;
  }
  return "long ago";
}
