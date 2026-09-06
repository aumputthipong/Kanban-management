# 0006 — Comment budget by block size, not density

**Status:** Accepted

## Context

Reading this codebase had become slow: opening a file often meant scrolling past
a paragraph of prose before reaching the first line of code. The obvious framing
was "too many comments — cut the percentage down".

Measuring it did not support that framing:

| | comment / non-blank |
|---|---|
| Backend Go (excl. generated + tests) | 15.5% |
| Backend tests | 12.2% |
| Frontend `ts`/`tsx` | 7.3% |

A 10–15% target was already met. Chasing the percentage would have changed
almost nothing about the experience that prompted the work.

The distribution was the real finding — 272 comment blocks of 4 lines or more,
holding 1,846 lines, 52 of them 10 lines or longer. Thirty comment lines spread
one at a time across three hundred lines of code reads fine; the same thirty
lines in one block is a wall. Same percentage, different file to work in.

Two further facts shaped the decision:

- **248 of those lines are Swagger annotations.** They look like comments and
  count like comments, but they generate `backend/docs` — deleting them breaks
  the OpenAPI spec.
- **Duplication had already drifted.** The PATCH-semantics rule was written in
  three places, and the copy in `planning_dto.go` had come to contradict both
  AGENTS.md and the handler it described. A stale comment is worse than no
  comment: a reader who trusts it writes the wrong code.

There is also a cost that does not show up in any measurement. When every
function carries an introductory paragraph, readers learn that comments are
skippable — and then skip the one that would have stopped a bad change. Volume
does not just slow reading down, it devalues the comments that matter.

## Decision

Budget **comment blocks**, not comment density.

- Any block is **≤ 3 content lines**; a block documenting a trap may use **4**;
  a file may hold **at most one** 4-line block. Swagger annotations are exempt.
- **Content lines only.** `/**`, `*/` and bare `*` do not count — counting them
  would leave one or two usable lines and make the budget unusable. Content that
  fits on one line is written as a one-line `/** … */`.
- Over budget means the material **moves** to `docs/` or an ADR and the code
  keeps a one-line pointer. It never means deleting the information.
- Comments are sorted into four kinds — API doc, trap, design rationale,
  narration — with a fixed verdict each (keep / keep in place / move / delete).
  The full table lives in AGENTS.md, "Comment & doc conventions".
- **A trap stays at the line where the mistake would be made.** This is the one
  category that must not move to docs: the reader is in the code, not in the
  docs. "404 not 403 is anti-enumeration" earns its four lines forever.
- Density is retained as a drift check (backend ≤ 12%, frontend ≤ 8%), never as
  a goal.

Rejected: a percentage target (measures the wrong thing, as above) and a blanket
maximum with no exception for traps (would mechanically delete the highest-value
comments in the repo — exactly the ones that prevent outages).

## Consequences

- Roughly 900–1,100 lines leave the source across the follow-up PRs. Every
  removed block either lands in `docs/` or is recorded as redundant with
  something already written there; nothing is dropped unreviewed.
- `docs/ARCHITECTURE.md` and the ADRs become load-bearing rather than optional.
  If they rot, the code no longer explains itself — that is the trade accepted
  here.
- For an AI agent this is a net gain in visibility, not a loss: AGENTS.md is
  loaded into every session, while a comment is only seen if that file happens
  to be opened.
- The rule is enforced by a check in `make verify` rather than by review
  discipline — AGENTS.md already carried comment rules and the codebase drifted
  anyway. The check ships **after** the cleanup PRs, because it would fail
  against all 272 existing blocks on day one.
- Comment density is not what makes this codebase hardest to maintain. 34
  frontend files exceed the 200-line rule AGENTS.md already sets, one of them at
  548 lines. Splitting those is a larger and separate piece of work.
