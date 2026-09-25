// Lazy + env-gated: the SDK is only imported when NEXT_PUBLIC_SENTRY_DSN is set.

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

/** No-op without a DSN; never throws. */
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

export const sentryEnabled = Boolean(dsn);
