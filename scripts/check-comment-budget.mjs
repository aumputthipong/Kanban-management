#!/usr/bin/env node
// Enforces the comment budget from docs/adr/0006-comment-budget.md: no block over
// MAX_TRAP content lines, and at most one block of that size per file. Run by
// `make check-comments` and by CI.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

const MAX_BLOCK = 3;
const MAX_TRAP = 4;
const MAX_TRAP_BLOCKS_PER_FILE = 1;

const ROOTS = ["frontend/src", "backend"];
const SKIP_DIRS = new Set(["node_modules", ".next", ".git", "dist", "build"]);
// sqlc writes internal/db and swag writes backend/docs; neither is hand-edited.
const SKIP_PATHS = ["backend/internal/db", "backend/docs"];
const EXTS = [".go", ".ts", ".tsx"];

// Delimiters and blank continuation lines cost vertical space but carry nothing,
// so they do not count against the budget.
const NON_CONTENT = new Set(["/**", "*/", "*", "/*", "//"]);

function* walk(dir) {
  for (const entry of readdirSync(dir)) {
    if (SKIP_DIRS.has(entry)) continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) yield* walk(full);
    else if (EXTS.some((e) => entry.endsWith(e))) yield full;
  }
}

function blocksIn(lines) {
  const blocks = [];
  let run = [];
  let start = 0;
  let inBlockComment = false;

  const flush = () => {
    if (run.length) {
      // Swagger annotations are machine-readable; they generate backend/docs.
      const isSwagger = run.some((l) => l.replace(/^[/*\s]+/, "").startsWith("@"));
      const content = run.filter((l) => !NON_CONTENT.has(l));
      if (!isSwagger && content.length > MAX_BLOCK) blocks.push({ start, size: content.length });
    }
    run = [];
  };

  lines.forEach((raw, i) => {
    const line = raw.trim();
    const isComment = inBlockComment || line.startsWith("//") || line.startsWith("*") || line.startsWith("/*");
    if (line.startsWith("/*") && !line.includes("*/")) inBlockComment = true;
    if (inBlockComment && line.includes("*/")) inBlockComment = false;
    if (isComment) {
      if (!run.length) start = i + 1;
      run.push(line);
    } else flush();
  });
  flush();
  return blocks;
}

const violations = [];
for (const root of ROOTS) {
  for (const file of walk(root)) {
    const rel = relative(process.cwd(), file).split(sep).join("/");
    if (SKIP_PATHS.some((p) => rel.startsWith(p))) continue;

    const oversized = blocksIn(readFileSync(file, "utf8").split(/\r?\n/));
    for (const b of oversized.filter((b) => b.size > MAX_TRAP)) {
      violations.push(`${rel}:${b.start}  ${b.size} content lines (max ${MAX_TRAP})`);
    }
    const traps = oversized.filter((b) => b.size === MAX_TRAP);
    if (traps.length > MAX_TRAP_BLOCKS_PER_FILE) {
      const at = traps.map((b) => b.start).join(", ");
      violations.push(`${rel}  ${traps.length} blocks of ${MAX_TRAP} lines at ${at} (max ${MAX_TRAP_BLOCKS_PER_FILE} per file)`);
    }
  }
}

if (violations.length) {
  console.error(`Comment budget exceeded in ${violations.length} place(s):\n`);
  for (const v of violations) console.error(`  ${v}`);
  console.error(`\nA block over ${MAX_BLOCK} lines needs to justify itself as a trap, and`);
  console.error(`anything longer moves to docs with a one-line pointer.`);
  console.error(`See AGENTS.md "Comment & doc conventions" and docs/adr/0006-comment-budget.md.`);
  process.exit(1);
}
console.log("Comment budget OK.");
