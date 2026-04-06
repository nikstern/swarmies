---
name: n
description: Start a Swarmies Beads session via a deterministic script that claims the first ready issue and begins implementation automatically.
---

# N

Use this skill when the user wants a fast, deterministic Beads session
bootstrap in the Swarmies repo.

## Contract

- Use `python3 plugins/n/scripts/start_session.py` as the source of truth.
- Do not reimplement task selection logic in the model.
- Claim the selected issue immediately and continue into implementation unless the script reports no ready work.
- If the script reports no ready work, stop there.

## Fixed flow

1. Run `bd prime` if session context needs refreshing.
2. Run `python3 plugins/n/scripts/start_session.py --auto-claim`.
3. Briefly report the claimed issue and begin implementation without waiting for another confirmation.

## Output format

- Show the chosen issue ID and title.
- State that it was selected by the script from the first `bd ready` result.
- Keep the summary short, note that the issue was auto-claimed, and proceed directly into implementation.

## Repo conventions

- Use `bd` for task tracking.
- For Go work, use `make test` and `make build` instead of raw `go test` or
  `go build`.
- End implementation sessions with `bd dolt push`, `git pull --rebase`, and
  `git push`.
