# Swarmies

An agent orchestration framework using:

- Go
- [Beads for task management](./docs/architecture/beads.md)
- [Planned tech-stack](./docs/tech-stack.md)

Current Plan:

- [v1](./docs/roadmap/v1/v1.md)

## Task management overview

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work atomically
bd close <id>         # Complete work
```
