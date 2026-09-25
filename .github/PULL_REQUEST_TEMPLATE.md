## Summary

<!-- 1–3 bullets: what changed and why. Skip the "what" if the diff is small. -->

## Test plan

<!-- How to verify this locally. Bulleted checklist. -->

- [ ] `make verify` passes locally (go vet + go test + tsc + vitest)
- [ ] For UI changes: opened the affected page in a browser and exercised the golden path
- [ ] For backend changes: covered new logic with a test

## Checklist

- [ ] Branch name follows the prefix in [`CONTRIBUTING.md`](../CONTRIBUTING.md#branch-naming) (`feat/`, `fix/`, `refactor/`, etc.)
- [ ] CI is green
- [ ] Docs updated if behaviour changed (`docs/`, `docs/CODE-NOTES.md`, `README`)
- [ ] Migration files included if the schema changed — additive forward (see [`docs/DATABASE.md`](../docs/DATABASE.md))
- [ ] OpenAPI spec regenerated (`make swag`) if handler annotations changed
- [ ] sqlc regenerated (`make sqlc`) and committed together with query changes
- [ ] No new ESLint warnings introduced
- [ ] No secrets committed (`.env`, credentials)

## Notes for reviewers

<!-- Anything worth flagging: trade-offs, follow-ups, screenshots for UI, etc. -->
