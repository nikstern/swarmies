---
name: n
description: Start a Swarmies Beads session quickly by inspecting ready work, selecting the best next issue, and claiming it before implementation.
---

# N

Use this skill when the user wants a fast Beads session bootstrap in the
Swarmies repo.

## Workflow

1. Run `bd prime` if session context needs refreshing.
2. Run `bd ready` to list claimable work.
3. Inspect the most relevant ready issues with `bd show <id>`.
4. Recommend the best next issue based on dependencies and readiness.
5. If the user says to proceed, run `bd update <id> --claim`.

## Repo conventions

- Use `bd` for task tracking.
- For Go work, use `make test` and `make build` instead of raw `go test` or
  `go build`.
- End implementation sessions with `bd dolt push`, `git pull --rebase`, and
  `git push`.
