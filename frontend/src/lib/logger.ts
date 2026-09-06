/**
 * Use instead of `console.*` so production output has one switch. `error` and `warn`
 * always log; `info` and `debug` are silenced in production builds unless
 * NEXT_PUBLIC_LOG_LEVEL=debug is set.
 */

const isProduction = process.env.NODE_ENV === "production";
const verbose = process.env.NEXT_PUBLIC_LOG_LEVEL === "debug";

const noop = () => {};

export const logger = {
  debug: isProduction && !verbose ? noop : (...args: unknown[]) => console.debug(...args),
  info: isProduction && !verbose ? noop : (...args: unknown[]) => console.info(...args),
  warn: (...args: unknown[]) => console.warn(...args),
  error: (...args: unknown[]) => console.error(...args),
};
