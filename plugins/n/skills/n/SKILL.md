---
name: n
description: Start a Swarmies Beads session via a deterministic script that previews the first ready issue and claims it on confirmation.
---

# N

Use this skill when the user wants a fast, deterministic Beads session
bootstrap in the Swarmies repo.

## Contract

- Use `python3 plugins/n/scripts/start_session.py` as the source of truth.
- Do not reimplement task selection logic in the model.
- Claim only after the user explicitly says to proceed.
- If the script reports no ready work, stop there.

## Fixed flow

1. Run `bd prime` if session context needs refreshing.
2. Run `python3 plugins/n/scripts/start_session.py`.
3. Present the script output briefly.
4. If the user says yes, run `python3 plugins/n/scripts/start_session.py --claim <id>`.

## Output format

- Show the chosen issue ID and title.
- State that it was selected by the script from the first `bd ready` result.
- Keep the summary short and follow the script output.

## Repo conventions

- Use `bd` for task tracking.
- For Go work, use `make test` and `make build` instead of raw `go test` or
  `go build`.
- End implementation sessions with `bd dolt push`, `git pull --rebase`, and
  `git push`.
