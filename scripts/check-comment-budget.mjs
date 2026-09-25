#!/usr/bin/env node
// Enforces docs/adr/0011-minimal-comments.md: no comment block over MAX_BLOCK
// content lines. Run by `make check-comments` and by CI.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

const MAX_BLOCK = 2;

const ROOTS = ["frontend/src", "backend"];
const SKIP_DIRS = new Set(["node_modules", ".next", ".git", "dist", "build"]);
// sqlc writes internal/db and swag writes backend/docs; neither is hand-edited.
const SKIP_PATHS = ["backend/internal/db", "backend/docs"];
const EXTS = [".go", ".ts", ".tsx"];

const NON_CONTENT = new Set(["/**", "*/", "*", "/*", "//", "{/*", "*/}"]);
const DIRECTIVE = /^(\/\/\s*(go:|nolint|eslint-|@ts-)|\/\*\s*eslint-)/;

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
      // Swagger annotations generate backend/docs.
      const isSwagger = run.some((l) => l.replace(/^[/*\s]+/, "").startsWith("@"));
      const content = run.filter((l) => !NON_CONTENT.has(l) && !DIRECTIVE.test(l));
      if (!isSwagger && content.length > MAX_BLOCK) blocks.push({ start, size: content.length });
    }
    run = [];
  };

  lines.forEach((raw, i) => {
    const line = raw.trim();
    const isComment =
      inBlockComment || line.startsWith("//") || line.startsWith("*") || line.startsWith("/*") || line.startsWith("{/*");
    if ((line.startsWith("/*") || line.startsWith("{/*")) && !line.includes("*/")) inBlockComment = true;
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
    for (const b of blocksIn(readFileSync(file, "utf8").split(/\r?\n/))) {
      violations.push(`${rel}:${b.start}  ${b.size} content lines (max ${MAX_BLOCK})`);
    }
  }
}

if (violations.length) {
  console.error(`Comment budget exceeded in ${violations.length} place(s):\n`);
  for (const v of violations) console.error(`  ${v}`);
  console.error(`\nMove the explanation to docs/CODE-NOTES.md or an ADR; keep at most a one-line pointer.`);
  console.error(`See AGENTS.md "Comment & doc conventions" and docs/adr/0011-minimal-comments.md.`);
  process.exit(1);
}
console.log("Comment budget OK.");
