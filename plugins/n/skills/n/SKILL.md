---
name: n
description: Start a Swarmies Beads session with a fixed flow: list ready work, inspect the first ready issue, and claim it on confirmation.
---

# N

Use this skill when the user wants a fast, deterministic Beads session
bootstrap in the Swarmies repo.

## Contract

- Run a fixed startup sequence.
- Do not invent ranking logic beyond Beads ordering.
- Treat the first issue returned by `bd ready` as the next issue.
- Claim only after the user explicitly says to proceed.
- If `bd ready` returns nothing, stop and report that no work is ready.

## Fixed flow

1. Run `bd prime` if session context needs refreshing.
2. Run `bd ready` to list claimable work.
3. Select the first issue in the `bd ready` output.
4. Run `bd show <id>` for that issue only.
5. Summarize the issue briefly and ask whether to claim it.
6. If the user says yes, run `bd update <id> --claim`.

## Output format

- Show the chosen issue ID and title.
- State that it was selected because it was first in `bd ready`.
- Keep the summary short: why, what, blockers or dependencies, and next action.

## Repo conventions

- Use `bd` for task tracking.
- For Go work, use `make test` and `make build` instead of raw `go test` or
  `go build`.
- End implementation sessions with `bd dolt push`, `git pull --rebase`, and
  `git push`.
