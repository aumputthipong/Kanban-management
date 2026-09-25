"use client";

import { useEffect } from "react";
import { logger } from "@/lib/logger";
import { captureException } from "@/lib/sentry";

// Replaces the root layout on crash, so it renders its own <html> and <body>.
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    logger.error("[global-error]", error);
    captureException(error, { boundary: "global", digest: error.digest });
  }, [error]);

  return (
    <html lang="th">
      <body
        style={{
          margin: 0,
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily:
            "system-ui, -apple-system, Segoe UI, Roboto, sans-serif",
          background: "#f8fafc",
          color: "#0f172a",
        }}
      >
        <div
          style={{
            maxWidth: 420,
            width: "100%",
            margin: 24,
            padding: 24,
            background: "white",
            border: "1px solid #fecdd3",
            borderRadius: 16,
            textAlign: "center",
          }}
        >
          <h1 style={{ margin: 0, fontSize: 20, fontWeight: 700 }}>
            แอปพบข้อผิดพลาดร้ายแรง
          </h1>
          <p style={{ marginTop: 8, color: "#64748b", fontSize: 14 }}>
            กรุณารีโหลดหน้านี้ ถ้ายังพบปัญหาให้ติดต่อทีมพัฒนา
          </p>
          {error.digest && (
            <p
              style={{
                marginTop: 12,
                fontFamily: "monospace",
                fontSize: 11,
                color: "#94a3b8",
              }}
            >
              ref: {error.digest}
            </p>
          )}
          <button
            type="button"
            onClick={reset}
            style={{
              marginTop: 16,
              padding: "8px 16px",
              background: "#2563eb",
              color: "white",
              border: "none",
              borderRadius: 8,
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            รีโหลด
          </button>
        </div>
      </body>
    </html>
  );
}
