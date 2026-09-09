#!/usr/bin/env node
// Keeps the WebSocket event contract honest across the Go/TypeScript boundary.
// Nothing else does: a mismatched tag compiles on both sides, passes every test,
// and only shows up as a board that quietly stops updating. See docs/adr/0007.
import { readFileSync } from "node:fs";
import { readdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const GO_CONSTANTS = join(root, "backend/internal/core/wsevent.go");
const TS_ENUM = join(root, "frontend/src/types/wsEvents.ts");
const HANDLER_DIR = join(root, "backend/internal/handler");

const problems = [];

// ── 1. The two event sets must be identical ──────────────────────────────────
const goEvents = new Set(
  [...readFileSync(GO_CONSTANTS, "utf8").matchAll(/WSEvent\s*=\s*"([A-Z_]+)"/g)].map((m) => m[1]),
);
const tsEvents = new Set(
  [...readFileSync(TS_ENUM, "utf8").matchAll(/^\s*\w+:\s*"([A-Z_]+)"/gm)].map((m) => m[1]),
);

if (goEvents.size === 0) problems.push(`no constants found in ${GO_CONSTANTS}`);
if (tsEvents.size === 0) problems.push(`no entries found in ${TS_ENUM}`);

for (const e of goEvents) {
  if (!tsEvents.has(e)) problems.push(`${e} is broadcast by Go but absent from wsEvents.ts`);
}
for (const e of tsEvents) {
  if (!goEvents.has(e)) problems.push(`${e} is declared in wsEvents.ts but no Go constant emits it`);
}

// ── 2. No handler may pass a raw string where a constant belongs ─────────────
for (const file of readdirSync(HANDLER_DIR).filter((f) => f.endsWith(".go") && !f.endsWith("_test.go"))) {
  const lines = readFileSync(join(HANDLER_DIR, file), "utf8").split("\n");
  lines.forEach((line, i) => {
    const m = line.match(/\bemit(?:To)?\([^)]*?,\s*"([A-Z_]+)"/);
    if (m) problems.push(`${file}:${i + 1} passes the literal "${m[1]}" — use a core.WS* constant`);
  });
}

if (problems.length > 0) {
  console.error(`WebSocket event contract broken in ${problems.length} place(s):\n`);
  for (const p of problems) console.error(`  ${p}`);
  console.error(
    "\nEvery broadcast tag needs a core.WS* constant and a matching entry in\n" +
      "frontend/src/types/wsEvents.ts. Adding one without the other leaves a\n" +
      "listener waiting for a message that never arrives.",
  );
  process.exit(1);
}

console.log(`WebSocket event contract OK (${goEvents.size} events).`);
