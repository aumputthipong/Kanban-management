// Use instead of console.*; info/debug are silent in production unless NEXT_PUBLIC_LOG_LEVEL=debug.

const isProduction = process.env.NODE_ENV === "production";
const verbose = process.env.NEXT_PUBLIC_LOG_LEVEL === "debug";

const noop = () => {};

export const logger = {
  debug: isProduction && !verbose ? noop : (...args: unknown[]) => console.debug(...args),
  info: isProduction && !verbose ? noop : (...args: unknown[]) => console.info(...args),
  warn: (...args: unknown[]) => console.warn(...args),
  error: (...args: unknown[]) => console.error(...args),
};
