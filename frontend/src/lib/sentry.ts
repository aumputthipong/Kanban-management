/**
 * Sentry browser integration — lazy, env-gated, no-op without a DSN. The SDK costs
 * ~30 KB, so without NEXT_PUBLIC_SENTRY_DSN it is never imported and the first
 * captureException call is what initialises it.
 */

import { logger } from "@/lib/logger";

const dsn = process.env.NEXT_PUBLIC_SENTRY_DSN;
const environment =
  process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ?? process.env.NODE_ENV ?? "development";
const release = process.env.NEXT_PUBLIC_SENTRY_RELEASE;

type SentryModule = typeof import("@sentry/browser");

let cached: Promise<SentryModule | null> | null = null;

function load(): Promise<SentryModule | null> {
  if (!dsn) return Promise.resolve(null);
  if (cached) return cached;
  cached = import("@sentry/browser")
    .then((mod) => {
      mod.init({
        dsn,
        environment,
        release,
        tracesSampleRate: 0,
      });
      return mod;
    })
    .catch((err) => {
      logger.error("[sentry] init failed", err);
      return null;
    });
  return cached;
}

/**
 * Report an unexpected error. No-op without a DSN, safe from any boundary, never throws.
 */
export function captureException(err: unknown, context?: Record<string, unknown>): void {
  if (!dsn) return;
  load().then((Sentry) => {
    if (!Sentry) return;
    if (context) {
      Sentry.withScope((scope) => {
        scope.setExtras(context);
        Sentry.captureException(err);
      });
    } else {
      Sentry.captureException(err);
    }
  });
}

/** Whether Sentry will actually ship events — for conditional "report this" UI. */
export const sentryEnabled = Boolean(dsn);
