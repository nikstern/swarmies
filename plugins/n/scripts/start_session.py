#!/usr/bin/env python3

import argparse
import json
import subprocess
import sys
from typing import Any


def run_bd(*args: str) -> str:
    result = subprocess.run(
        ["bd", *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout


def run_bd_json(*args: str) -> Any:
    return json.loads(run_bd(*args, "--json"))


def first_paragraph_by_heading(text: str, heading: str) -> str:
    marker = f"{heading}:"
    start = text.find(marker)
    if start == -1:
        return ""

    remainder = text[start + len(marker) :].lstrip()
    parts = remainder.split("\n\n", 1)
    return parts[0].strip()


def preview_first_ready() -> int:
    ready = run_bd_json("ready", "--limit", "1")
    if not ready:
        print("status: empty")
        print("message: no ready work")
        return 0

    issue = ready[0]
    issue_id = issue["id"]
    detail = run_bd_json("show", issue_id)[0]

    why = first_paragraph_by_heading(detail.get("description", ""), "Why")
    what = first_paragraph_by_heading(detail.get("description", ""), "What")
    dependencies = detail.get("dependencies", [])
    blocked_by = [dep["id"] for dep in dependencies if dep.get("status") != "closed"]

    print("status: ready")
    print(f"id: {issue_id}")
    print(f"title: {detail.get('title', '')}")
    print("selection_rule: first issue returned by `bd ready --json --limit 1`")
    if why:
        print(f"why: {why}")
    if what:
        print(f"what: {what}")
    if blocked_by:
        print(f"open_dependencies: {', '.join(blocked_by)}")
    else:
        print("open_dependencies: none")
    print(f"next: python3 plugins/n/scripts/start_session.py --claim {issue_id}")
    return 0


def claim(issue_id: str) -> int:
    run_bd("update", issue_id, "--claim")
    print("status: claimed")
    print(f"id: {issue_id}")
    print(f"next: bd show {issue_id}")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Deterministic Beads session bootstrap for the n plugin."
    )
    parser.add_argument("--claim", metavar="ISSUE_ID", help="Claim a specific issue ID")
    args = parser.parse_args()

    try:
        if args.claim:
            return claim(args.claim)
        return preview_first_ready()
    except subprocess.CalledProcessError as exc:
        stderr = exc.stderr.strip()
        if stderr:
            print(stderr, file=sys.stderr)
        return exc.returncode


if __name__ == "__main__":
    raise SystemExit(main())
