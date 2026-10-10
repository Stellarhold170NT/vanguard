#!/usr/bin/env python3
"""factory-metrics.py — measure the VANGUARD software factory from its own records.

Reads two data sources and emits machine-readable metrics + a markdown table:

  1. Task contracts  (taskloop/task-*.task.json)   — CURRENT status snapshots.
  2. Factory runlog  (state/runlog.jsonl)          — append-only event log written
     by the orchestrator (PROPOSED event schema in docs/factory-metrics.md).

Outputs (paths configurable, defaults relative to the current directory):
  --out  state/factory-metrics.json   schema_version=1, stable keys (consumed by w7-04)
  --md   state/factory-metrics.md     human-readable tables (same numbers, same source)

Design rules (factory protocol):
  * runlog.jsonl is source of truth — this script never writes to it;
    if a metric looks wrong, fix THIS script, not the log.
  * Missing data is reported as null + an explicit "available" flag — never
    silently replaced by 0 (a fake zero is worse than an honest null).

Exit codes: 0 = metrics written; 2 = input error (contracts dir missing/empty).
Stdlib only (no third-party deps) so it runs in the dind-sandbox after one
`apk add python3`.

Metric definitions (formulas) live in docs/factory-metrics.md and are also
embedded in the JSON output under "definitions" so the JSON is self-describing.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

SCHEMA_VERSION = 1
TERMINAL_STATUSES = {"DONE", "DONE_WITH_CONCERNS", "BLOCKED", "NEEDS_CONTEXT"}
RUNLOG_TERMINAL_STATUSES = TERMINAL_STATUSES  # runlog "to" values that close an arm

# Tolerant key aliases: the runlog format is proposed, not yet observed, so the
# parser accepts common aliases and ignores lines it cannot understand (counted,
# never fatal).
TS_KEYS = ("ts", "timestamp", "time", "@timestamp", "at")
EVENT_KEYS = ("event", "type", "action", "kind")
TASK_KEYS = ("task", "task_id", "taskid", "task_key", "name", "id")
FROM_KEYS = ("from", "from_status", "prev", "previous")
TO_KEYS = ("to", "to_status", "status", "new")
WORKFLOW_RE = re.compile(r"\[w(\d+)-(\d+)\]", re.IGNORECASE)
GATE_DECISION_RE = re.compile(r"^\s*(APPROVED|REJECTED)\b", re.IGNORECASE)


def parse_iso(ts: str):
    """Parse an ISO-8601 timestamp (Z or offset) → aware datetime, else None."""
    if not isinstance(ts, str) or not ts.strip():
        return None
    s = ts.strip()
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    try:
        dt = datetime.fromisoformat(s)
    except ValueError:
        return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt


def iso_z(dt):
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ") if dt else None


def hours_between(a, b):
    if not a or not b:
        return None
    return round((b - a).total_seconds() / 3600.0, 3)


def first_of(d, keys):
    for k in keys:
        if k in d and d[k] is not None:
            return d[k]
    return None


# --------------------------------------------------------------------------- #
# Contracts (current status snapshots)
# --------------------------------------------------------------------------- #

def load_contracts(contracts_dir: Path):
    """Parse taskloop/task-*.task.json.

    Returns (contracts, parse_errors, dir_available) where contracts is a list
    of normalized contract dicts."""
    contracts, parse_errors = [], []
    if not contracts_dir.is_dir():
        return contracts, parse_errors, False
    files = sorted(
        contracts_dir.glob("task-*.task.json"),
        key=lambda p: (len(p.stem), p.stem),  # numeric-ish order: task-2 < task-10
    )
    for path in files:
        try:
            raw = json.loads(path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError, OSError) as exc:
            parse_errors.append({"file": path.name, "error": str(exc)[:200]})
            continue
        n = str(raw.get("n", "")).strip()
        name = str(raw.get("name", "")).strip()
        m = WORKFLOW_RE.search(name)
        contracts.append(
            {
                "n": n,
                "name": name,
                "workflow": ("w%s" % m.group(1)) if m else None,
                "task_id": ("w%s-%s" % (m.group(1), m.group(2))) if m else n,
                "model": raw.get("model"),
                "status": str(raw.get("status", "UNKNOWN")),
                "brief": raw.get("brief"),
                "report": raw.get("report"),
                "dir": raw.get("dir"),
                "updated_at": parse_iso(raw.get("updatedAt")),
                "comments": raw.get("comments") or [],
            }
        )
    return contracts, parse_errors, True


def contract_gate_events(contracts):
    """Gate decisions recorded as contract comments (author contains 'gate')."""
    events = []
    for c in contracts:
        for cm in c["comments"]:
            author = str(cm.get("author", ""))
            if "gate" not in author.lower():
                continue
            text = str(cm.get("text", ""))
            decision_m = GATE_DECISION_RE.search(text)
            events.append(
                {
                    "task": c["task_id"],
                    "task_n": c["n"],
                    "timestamp": iso_z(parse_iso(cm.get("timestamp"))),
                    "author": author,
                    "decision": decision_m.group(1).upper() if decision_m else "OTHER",
                    "text": text[:500],
                    "source": "contract_comment",
                }
            )
    events.sort(key=lambda e: (e["timestamp"] or "", str(e["task_n"]).zfill(4)))
    return events


def summarize_contracts(contracts):
    """Aggregate the current-status snapshot by workflow."""
    workflows = {}
    for c in contracts:
        wf = workflows.setdefault(
            c["workflow"] or "unknown",
            {
                "workflow": c["workflow"] or "unknown",
                "tasks_total": 0,
                "status_counts": {},
                "done": 0,
                "done_with_concerns": 0,
                "blocked": 0,
                "needs_context": 0,
                "in_progress": 0,
                "pending": 0,
                "first_touch": None,
                "last_touch": None,
            },
        )
        wf["tasks_total"] += 1
        st = c["status"]
        wf["status_counts"][st] = wf["status_counts"].get(st, 0) + 1
        if st == "DONE":
            wf["done"] += 1
        elif st == "DONE_WITH_CONCERNS":
            wf["done_with_concerns"] += 1
        elif st == "BLOCKED":
            wf["blocked"] += 1
        elif st == "NEEDS_CONTEXT":
            wf["needs_context"] += 1
        elif st == "IN_PROGRESS":
            wf["in_progress"] += 1
        elif st == "PENDING":
            wf["pending"] += 1
        # first_touch: a PENDING contract was never touched after creation, so
        # its updatedAt == creation; for others updatedAt == last touch.
        if c["updated_at"] and (wf["first_touch"] is None or c["updated_at"] < wf["first_touch"]):
            wf["first_touch"] = c["updated_at"]
        if c["updated_at"] and (wf["last_touch"] is None or c["updated_at"] > wf["last_touch"]):
            wf["last_touch"] = c["updated_at"]

    for wf in workflows.values():
        # span is a proxy over contract timestamps, NOT a per-task duration
        # (normalize datetimes → ISO strings BEFORE anything serializes them)
        wf["span_hours"] = hours_between(wf["first_touch"], wf["last_touch"])
        wf["first_touch"] = iso_z(wf["first_touch"])
        wf["last_touch"] = iso_z(wf["last_touch"])
        finished = wf["done"] + wf["done_with_concerns"]
        wf["concerns_rate"] = (
            round(wf["done_with_concerns"] / finished, 4) if finished else None
        )

    # report-file existence check: contract 'report' paths are relative to 'dir'.
    # Performed only when the contract 'dir' roots are visible on THIS host —
    # otherwise the check would report every report as "missing" (a false
    # alarm), e.g. when the script runs inside a sandbox without the workspace.
    with_dir = [c for c in contracts if c["report"] and c["dir"]]
    dirs_visible = sum(1 for c in with_dir if Path(c["dir"]).is_dir())
    reports_check = {"performed": False, "dirs_visible": dirs_visible,
                     "dirs_total": len(with_dir),
                     "reason": None, "missing": []}
    if with_dir and dirs_visible == len(with_dir):
        reports_check["performed"] = True
        for c in with_dir:
            if not (Path(c["dir"]) / c["report"]).exists():
                reports_check["missing"].append({"task": c["task_id"], "report": c["report"]})
    elif with_dir:
        reports_check["reason"] = (
            "contract 'dir' paths not present on this host (%d/%d visible) — "
            "existence check skipped; run from the factory workspace to enable it"
            % (dirs_visible, len(with_dir))
        )

    out = []
    for key in sorted(workflows, key=lambda k: (len(k), k)):
        wf = workflows[key]
        wf["avg_task_hours"] = None   # per-task duration needs the runlog
        wf["min_task_hours"] = None
        wf["max_task_hours"] = None
        wf["rearm_count"] = None
        out.append(wf)

    total_done = sum(w["done"] for w in out)
    total_concerns = sum(w["done_with_concerns"] for w in out)
    total_finished = total_done + total_concerns
    gate_events = contract_gate_events(contracts)
    models = {}
    for c in contracts:
        models[str(c["model"])] = models.get(str(c["model"]), 0) + 1

    cycle_first = min((c["updated_at"] for c in contracts if c["updated_at"]), default=None)
    cycle_last = max((c["updated_at"] for c in contracts if c["updated_at"]), default=None)

    totals = {
        "contracts_parsed": len(contracts),
        "tasks_total": sum(w["tasks_total"] for w in out),
        "done": total_done,
        "done_with_concerns": total_concerns,
        "concerns_rate": round(total_concerns / total_finished, 4) if total_finished else None,
        "in_progress": sum(w["in_progress"] for w in out),
        "pending": sum(w["pending"] for w in out),
        "blocked": sum(w["blocked"] for w in out),
        "needs_context": sum(w["needs_context"] for w in out),
        "gates_approved": sum(1 for e in gate_events if e["decision"] == "APPROVED"),
        "gates_rejected": sum(1 for e in gate_events if e["decision"] == "REJECTED"),
        "reports_missing_count": len(reports_check["missing"]),
        "first_touch": iso_z(cycle_first),
        "last_touch": iso_z(cycle_last),
        "span_hours": hours_between(cycle_first, cycle_last),
        "models": models,
    }
    return {
        "workflows": out,
        "totals": totals,
        "gate_events": gate_events,
        "reports_check": reports_check,
    }


# --------------------------------------------------------------------------- #
# Runlog (append-only event log; PROPOSED schema, tolerant parsing)
# --------------------------------------------------------------------------- #

def load_runlog(runlog_path: Path):
    """Parse state/runlog.jsonl. Returns (events, meta). Missing file → available=False."""
    meta = {
        "available": False,
        "path": str(runlog_path),
        "lines_total": 0,
        "events_parsed": 0,
        "lines_unrecognized": 0,
        "parse_errors": [],
    }
    if not runlog_path.is_file():
        return [], meta
    meta["available"] = True
    events = []
    with runlog_path.open("r", encoding="utf-8", errors="replace") as fh:
        for i, line in enumerate(fh, 1):
            line = line.strip()
            meta["lines_total"] += 1
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError as exc:
                meta["parse_errors"].append({"line": i, "error": str(exc)[:200]})
                meta["lines_unrecognized"] += 1
                continue
            if not isinstance(obj, dict):
                meta["lines_unrecognized"] += 1
                continue
            ts = parse_iso(first_of(obj, TS_KEYS) or "")
            ev = str(first_of(obj, EVENT_KEYS) or "").lower()
            task = first_of(obj, TASK_KEYS)
            if not ts or not ev or task is None:
                meta["lines_unrecognized"] += 1
                continue
            events.append(
                {
                    "timestamp": ts,
                    "event": ev,
                    "task": str(task),
                    "from": first_of(obj, FROM_KEYS),
                    "to": first_of(obj, TO_KEYS),
                    "raw": obj,
                }
            )
            meta["events_parsed"] += 1
    events.sort(key=lambda e: e["timestamp"])
    return events, meta


def canonical_task_id(task, n_map):
    """Normalize a runlog task id to the canonical 'wN-xx' form.

    Accepts 'w7-03', '[w7-03] Factory metrics dashboard' (contract name form),
    'task-39' and bare contract numbers ('39'). Unknown ids are returned
    trimmed so they can be REPORTED as unmapped instead of fragmenting arms
    under different spellings of the same task.
    """
    t = str(task)
    m = WORKFLOW_RE.search(t)
    if m:
        return "w%s-%s" % (m.group(1), m.group(2))
    key = t.strip().lower()
    if key in n_map:
        return n_map[key]
    if key.startswith("task-") and key[5:] in n_map:
        return n_map[key[5:]]
    return t.strip()


def arm_analysis(events, n_map=None):
    """Group status events into task 'arms' (PENDING→IN_PROGRESS→terminal).

    Task ids are canonicalized first (canonical_task_id) so the same task
    logged under '[w7-03] …', 'task-39' and '39' forms ONE arm history.
    An arm opens at a transition into IN_PROGRESS and closes at the next
    transition into a terminal status. A task with >1 arms was RE-ARMED
    (re-dispatched after a terminal status). This is the runlog-derived,
    objective definition of re-arm; contracts alone cannot show it.
    A terminal event arriving with no open arm is counted as an orphan
    (log started mid-task, or duplicate terminal) — never fabricated
    into a duration.
    """
    n_map = n_map or {}
    by_task = {}
    orphan_terminal_events = 0
    for e in events:
        if e["event"] in ("task_status", "status", "status_change", "task_update"):
            task = canonical_task_id(e["task"], n_map)
            by_task.setdefault(task, []).append(e)

    arms = []
    for task, evs in by_task.items():
        open_start = None
        for e in evs:
            to = str(e["to"] or "").upper()
            if to == "IN_PROGRESS" and open_start is None:
                open_start = e["timestamp"]
            elif to in RUNLOG_TERMINAL_STATUSES:
                if open_start is None:
                    orphan_terminal_events += 1
                    continue
                arms.append(
                    {
                        "task": task,
                        "started": iso_z(open_start),
                        "ended": iso_z(e["timestamp"]),
                        "hours": hours_between(open_start, e["timestamp"]),
                        "end_status": to,
                        "arm_index": len([a for a in arms if a["task"] == task]) + 1,
                    }
                )
                open_start = None
        if open_start is not None:  # still open at end of log
            arms.append(
                {
                    "task": task,
                    "started": iso_z(open_start),
                    "ended": None,
                    "hours": None,
                    "end_status": "OPEN",
                    "arm_index": len([a for a in arms if a["task"] == task]) + 1,
                }
            )

    per_task = {}
    for a in arms:
        per_task.setdefault(a["task"], []).append(a)
    rearm_counts = {t: max(0, len(v) - 1) for t, v in per_task.items()}
    first_try_done = [
        t
        for t, v in per_task.items()
        if len(v) >= 1 and v[0]["end_status"] == "DONE"
    ]
    done_tasks = [
        t
        for t, v in per_task.items()
        if any(a["end_status"] == "DONE" for a in v)
    ]
    durations = [a["hours"] for a in arms if a["hours"] is not None]
    return {
        "arms": arms,
        "per_task": per_task,
        "rearm_counts": rearm_counts,
        "tasks_measured": len(per_task),
        "tasks_rearmed": sum(1 for c in rearm_counts.values() if c > 0),
        "rearm_events_total": sum(rearm_counts.values()),
        "orphan_terminal_events": orphan_terminal_events,
        "done_first_try": len(first_try_done),
        "done_total": len(done_tasks),
        "done_first_try_rate": (
            round(len(first_try_done) / len(done_tasks), 4) if done_tasks else None
        ),
        "duration_hours": {
            "avg": round(sum(durations) / len(durations), 3) if durations else None,
            "min": min(durations) if durations else None,
            "max": max(durations) if durations else None,
            "count": len(durations),
        },
        "runlog_gate_events": [
            {
                "task": e["task"],
                "timestamp": iso_z(e["timestamp"]),
                "decision": str(e["to"] or "").upper(),
                "source": "runlog_event",
            }
            for e in events
            if e["event"] in ("gate_result", "gate_decision", "gate")
        ],
    }


def apply_runlog_to_workflows(workflows, runlog_summary, contracts=None):
    """Fill avg/min/max/rearm per workflow from runlog arms.

    Task ids in the runlog may be written as 'w7-03', '[w7-03] Factory…',
    'task-39' or a bare contract number ('39'). Resolution order: wN-xx regex
    first, then the contract n→workflow map; anything still unmapped is
    REPORTED (returned as unmapped_tasks), never silently guessed.
    """
    if not runlog_summary["tasks_measured"]:
        return {"unmapped_tasks": []}
    n_map = {}
    for c in contracts or []:
        if c.get("n") and c.get("workflow"):
            n_map[str(c["n"]).strip()] = c["workflow"]
            n_map["task-%s" % str(c["n"]).strip()] = c["workflow"]

    def task_workflow(task):
        m = re.search(r"w(\d+)-(\d+)", task)
        if m:
            return "w%s" % m.group(1)
        t = task.strip().lower()
        return n_map.get(t)

    per_wf = {}
    unmapped = []
    for task, arms in runlog_summary["per_task"].items():
        wf = task_workflow(task)
        if wf is None:
            unmapped.append(task)
            continue
        per_wf.setdefault(wf, []).extend(a["hours"] for a in arms if a["hours"] is not None)
        rearm = max(0, len(arms) - 1)
        for w in workflows:
            if w["workflow"] == wf:
                w["rearm_count"] = (w["rearm_count"] or 0) + rearm
    for w in workflows:
        durs = per_wf.get(w["workflow"]) or []
        if durs:
            w["avg_task_hours"] = round(sum(durs) / len(durs), 3)
            w["min_task_hours"] = min(durs)
            w["max_task_hours"] = max(durs)
    return {"unmapped_tasks": sorted(unmapped)}


# --------------------------------------------------------------------------- #
# Output
# --------------------------------------------------------------------------- #

DEFINITIONS = {
    "concerns_rate": (
        "DONE_WITH_CONCERNS / (DONE + DONE_WITH_CONCERNS), per workflow and total; "
        "null when no task has reached a DONE* terminal status. Source: contract statuses."
    ),
    "span_hours": (
        "last_touch - first_touch over contract updatedAt timestamps (first_touch of a "
        "PENDING contract == its creation; for finished tasks updatedAt == completion). "
        "This is a workflow WALL-CLOCK PROXY, not per-task duration. Source: contracts."
    ),
    "avg_min_max_task_hours": (
        "Per-task duration = terminal transition timestamp - first IN_PROGRESS transition "
        "timestamp of the same arm, in hours (mean/min/max per workflow). "
        "SOURCE: runlog.jsonl only — null while the runlog does not exist."
    ),
    "rearm_count": (
        "Number of arms beyond the first per task (a task re-dispatched after a terminal "
        "status adds 1 re-arm). SOURCE: runlog.jsonl only — null while the runlog is absent; "
        "0 would be a fake zero, so it is never substituted."
    ),
    "done_first_try_rate": (
        "Tasks whose FIRST arm ended DONE / all tasks that ever reached DONE. "
        "SOURCE: runlog.jsonl only."
    ),
    "gates": (
        "Gate decisions: contract comments whose author contains 'gate' (snapshot — may "
        "miss superseded comments) plus runlog gate_result events when present."
    ),
    "reports_missing_count": (
        "Contracts whose declared report file does not exist on disk at generation "
        "time. The check runs only when the contract 'dir' roots are visible on the "
        "host running the script (else 'performed'=false and the count is not "
        "meaningful). See 'reports_check' in the JSON."
    ),
}

PROPOSED_RUNLOG_SCHEMA = {
    "file": "state/runlog.jsonl",
    "writer": "orchestrator (append-only; this script is read-only)",
    "line_format": "one JSON object per line (JSONL)",
    "required_fields": {
        "ts": "ISO-8601 timestamp (Z or offset) — aliases: timestamp|time|@timestamp",
        "event": "event name, e.g. task_status — aliases: type|action|kind",
        "task": "task id, e.g. w7-03 — aliases: task_id|taskid|name|id",
        "from": "status before the transition (task_status events)",
        "to": "status after the transition (task_status events); "
        "terminal: DONE|DONE_WITH_CONCERNS|BLOCKED|NEEDS_CONTEXT",
    },
    "optional_events": [
        "gate_result / gate_decision / gate — fields: task, to=APPROVED|REJECTED",
        "unknown event names are counted in lines_unrecognized, never fatal",
    ],
    "example_line": '{"ts":"2026-10-10T06:35:11Z","event":"task_status","task":"w7-03","from":"PENDING","to":"IN_PROGRESS"}',
}


def build_markdown(payload):
    """Render the markdown table(s) from the SAME payload that produced the JSON."""
    lines = []
    t = payload["totals"]
    lines.append("# Factory metrics — VANGUARD (generated)\n")
    lines.append(
        "> Generated by `tools/factory-metrics.py` at %s (schema v%d). "
        "Metric definitions: `docs/factory-metrics.md`. This file is a generated "
        "snapshot — regenerate, do not hand-edit.\n" % (payload["generated_at"], SCHEMA_VERSION)
    )
    lines.append("## Data scope\n")
    src = payload["data_sources"]
    lines.append(
        "- Contracts: **%s** (%d files parsed, %d parse errors)"
        % (
            "AVAILABLE" if src["contracts"]["available"] else "MISSING",
            src["contracts"]["files_parsed"],
            len(src["contracts"]["parse_errors"]),
        )
    )
    lines.append(
        "- Runlog `%s`: **%s**"
        % (
            src["runlog"]["path"],
            ("AVAILABLE (%d events)" % src["runlog"]["events_parsed"])
            if src["runlog"]["available"]
            else (
                "NOT PRESENT — per-task durations, re-arm counts and first-try rates "
                "are `null` (not zero) until the orchestrator starts writing this log"
            ),
        )
    )
    lines.append(
        "- Contract-timestamp window: %s → %s (%.1f h span)\n"
        % (
            t["first_touch"] or "n/a",
            t["last_touch"] or "n/a",
            t["span_hours"] or 0.0,
        )
    )

    lines.append("## Per-workflow summary (from contract statuses)\n")
    lines.append(
        "| workflow | tasks | done | concerns | concerns_rate | in_prog | pending | span_h | avg_h | min_h | max_h | rearm |"
    )
    lines.append("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|")
    for w in payload["workflows"]:
        lines.append(
            "| %s | %d | %d | %d | %s | %d | %d | %s | %s | %s | %s | %s |"
            % (
                w["workflow"],
                w["tasks_total"],
                w["done"],
                w["done_with_concerns"],
                "%.1f%%" % (100 * w["concerns_rate"]) if w["concerns_rate"] is not None else "n/a",
                w["in_progress"],
                w["pending"],
                "%.1f" % w["span_hours"] if w["span_hours"] is not None else "n/a",
                "%.1f" % w["avg_task_hours"] if w["avg_task_hours"] is not None else "n/a",
                "%.1f" % w["min_task_hours"] if w["min_task_hours"] is not None else "n/a",
                "%.1f" % w["max_task_hours"] if w["max_task_hours"] is not None else "n/a",
                w["rearm_count"] if w["rearm_count"] is not None else "n/a",
            )
        )
    lines.append(
        "| **total** | %d | %d | %d | %s | %d | %d | %s | %s | %s | %s | %s |"
        % (
            t["tasks_total"],
            t["done"],
            t["done_with_concerns"],
            "%.1f%%" % (100 * t["concerns_rate"]) if t["concerns_rate"] is not None else "n/a",
            t["in_progress"],
            t["pending"],
            "%.1f" % t["span_hours"] if t["span_hours"] is not None else "n/a",
            "n/a",
            "n/a",
            "n/a",
            "n/a",
        )
    )
    lines.append("")
    lines.append(
        "Reads: `concerns_rate` = DONE_WITH_CONCERNS / (DONE + DONE_WITH_CONCERNS); "
        "`span_h` = wall-clock proxy between earliest and latest contract touch "
        "(NOT per-task duration); `avg/min/max_h` + `rearm` come from the runlog only.\n"
    )

    gates = payload["gates"]
    lines.append("## Gate history (%d events)\n" % len(gates["events"]))
    if gates["events"]:
        lines.append("| timestamp | task | decision | author | source |")
        lines.append("|---|---|---|---|---|")
        for g in gates["events"]:
            lines.append(
                "| %s | %s | %s | %s | %s |" % (g["timestamp"], g["task"], g["decision"], g["author"], g["source"])
            )
    else:
        lines.append("(no gate events found in either source)")
    lines.append("")

    lines.append("## Runlog-derived aggregates\n")
    rs = gates.get("runlog", {})
    if rs.get("tasks_measured"):
        lines.append("- tasks measured: %d (arms: %d)" % (rs["tasks_measured"], len(rs["arms"])))
        lines.append("- task duration avg/min/max: %s / %s / %s h" % (
            rs["duration_hours"]["avg"], rs["duration_hours"]["min"], rs["duration_hours"]["max"]))
        lines.append("- re-armed tasks: %d (re-arm events: %d)" % (rs["tasks_rearmed"], rs["rearm_events_total"]))
        lines.append("- DONE first-try rate: %s" % (
            "%.1f%%" % (100 * rs["done_first_try_rate"]) if rs["done_first_try_rate"] is not None else "n/a"))
    else:
        lines.append(
            "Runlog not available yet — `avg/min/max task hours`, `rearm_count`, "
            "`done_first_try_rate` are **null by design** (unknown ≠ 0). "
            "See `docs/factory-metrics.md` §'runlog event schema' for the proposed "
            "orchestrator log format that turns these on."
        )
    lines.append("")
    rc = payload["reports_check"]
    if rc["performed"] and rc["missing"]:
        lines.append(
            "⚠ %d contract(s) declare a report file that does not exist on disk: %s"
            % (len(rc["missing"]), ", ".join(m["task"] for m in rc["missing"]))
        )
    elif rc["performed"]:
        lines.append(
            "Report-file existence check: PASS — every contract's declared report exists (%d contracts with a report path)."
            % rc["dirs_total"]
        )
    else:
        lines.append(
            "Report-file existence check: SKIPPED — %s" % (rc["reason"] or "contract dirs not visible")
        )
    return "\n".join(lines) + "\n"


def main(argv=None):
    ap = argparse.ArgumentParser(
        description="Compute VANGUARD factory metrics from contracts + runlog."
    )
    ap.add_argument("--contracts", default="taskloop", help="directory of task-*.task.json contracts")
    ap.add_argument("--runlog", default="state/runlog.jsonl", help="path to runlog.jsonl")
    ap.add_argument("--out", default="state/factory-metrics.json", help="output JSON path")
    ap.add_argument("--md", default="state/factory-metrics.md", help="output markdown path")
    args = ap.parse_args(argv)

    contracts_dir = Path(args.contracts)
    runlog_path = Path(args.runlog)
    out_path = Path(args.out)
    md_path = Path(args.md)

    contracts, contract_errors, contracts_available = load_contracts(contracts_dir)
    if not contracts:
        sys.stderr.write(
            "factory-metrics: no contracts parsed from %s (available=%s, errors=%d) — "
            "nothing to measure; fix --contracts and retry\n"
            % (contracts_dir, contracts_available, len(contract_errors))
        )
        return 2

    runlog_events, runlog_meta = load_runlog(runlog_path)
    # contract-number → canonical task id, so bare numeric ids in the runlog
    # ('39') and 'task-39' spellings fold into the same arm history as 'w7-03'
    task_n_map = {
        str(c["n"]).strip().lower(): c["task_id"]
        for c in contracts
        if c["n"] and c["task_id"]
    }
    runlog_summary = arm_analysis(runlog_events, task_n_map) if runlog_meta["available"] else {
        "arms": [], "per_task": {}, "rearm_counts": {}, "tasks_measured": 0,
        "tasks_rearmed": 0, "rearm_events_total": 0, "orphan_terminal_events": 0,
        "done_first_try": 0,
        "done_total": 0, "done_first_try_rate": None,
        "duration_hours": {"avg": None, "min": None, "max": None, "count": 0},
        "runlog_gate_events": [],
    }

    summary = summarize_contracts(contracts)
    runlog_mapping = apply_runlog_to_workflows(summary["workflows"], runlog_summary, contracts) or {"unmapped_tasks": []}

    now = datetime.now(timezone.utc)
    gates_all = summary["gate_events"] + runlog_summary["runlog_gate_events"]
    gates_all.sort(key=lambda e: (e["timestamp"] or "", str(e.get("task"))))

    payload = {
        "schema_version": SCHEMA_VERSION,
        "generated_at": iso_z(now),
        "generator": {
            "script": "tools/factory-metrics.py",
            "args": {
                "contracts": str(contracts_dir),
                "runlog": str(runlog_path),
            },
        },
        "data_sources": {
            "contracts": {
                "available": contracts_available,
                "dir": str(contracts_dir),
                "files_parsed": len(contracts),
                "parse_errors": contract_errors,
            },
            "runlog": runlog_meta,
        },
        "totals": summary["totals"],
        "workflows": summary["workflows"],
        "gates": {
            "events": gates_all,
            "runlog": {
                "tasks_measured": runlog_summary["tasks_measured"],
                "tasks_rearmed": runlog_summary["tasks_rearmed"],
                "rearm_events_total": runlog_summary["rearm_events_total"],
                "done_first_try": runlog_summary["done_first_try"],
                "done_total": runlog_summary["done_total"],
                "done_first_try_rate": runlog_summary["done_first_try_rate"],
                "duration_hours": runlog_summary["duration_hours"],
                "arms": runlog_summary["arms"],
                "orphan_terminal_events": runlog_summary["orphan_terminal_events"],
                "unmapped_tasks": runlog_mapping["unmapped_tasks"],
            },
        },
        "reports_check": summary["reports_check"],
        "definitions": DEFINITIONS,
        "runlog_event_schema_proposed": PROPOSED_RUNLOG_SCHEMA,
    }

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, indent=2, sort_keys=False) + "\n", encoding="utf-8")

    md = build_markdown(payload)
    md_path.parent.mkdir(parents=True, exist_ok=True)
    md_path.write_text(md, encoding="utf-8")

    t = payload["totals"]
    sys.stdout.write(
        "factory-metrics: %d contracts, %d workflows, %d done, %d concerns "
        "(rate %s), %d gates approved, %d runlog events\n"
        "  → %s\n  → %s\n"
        % (
            t["contracts_parsed"],
            len(payload["workflows"]),
            t["done"],
            t["done_with_concerns"],
            ("%.1f%%" % (100 * t["concerns_rate"])) if t["concerns_rate"] is not None else "n/a",
            t["gates_approved"],
            runlog_meta["events_parsed"],
            out_path,
            md_path,
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
