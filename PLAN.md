# BSG — A Spec Graph CLI

## Context

Build a CLI tool called `bsg` that manages behavioral specifications as a dependency-aware graph backed by SQLite. Unlike task trackers (beads, linear, github issues), bsg tracks **what the system should do** — living specs that evolve, link to code, and can be verified. Each spec is a structured unit with typed relationships to other specs and traceability to source files.

Written in Go. New repo at `/Users/jasonmay/jml/bsg`.

## Data Model

### SQLite Schema

```sql
specs (
  id          TEXT PRIMARY KEY,    -- short hash: bsg-a3f2
  name        TEXT NOT NULL,
  type        TEXT NOT NULL,       -- behavior | constraint | interface | data-shape | invariant
  status      TEXT NOT NULL DEFAULT 'draft',  -- draft | accepted | implemented | verified | deprecated
  body        TEXT NOT NULL,
  tags        TEXT,                -- JSON array: ["weight","input"]
  created_at  TEXT NOT NULL,       -- RFC3339
  updated_at  TEXT NOT NULL
)

edges (
  from_id     TEXT NOT NULL REFERENCES specs(id),
  to_id       TEXT NOT NULL REFERENCES specs(id),
  relation    TEXT NOT NULL,       -- depends_on | refines | conflicts_with | implements | supersedes
  created_at  TEXT NOT NULL,
  PRIMARY KEY (from_id, to_id, relation)
)

code_links (
  spec_id     TEXT NOT NULL REFERENCES specs(id),
  file_path   TEXT NOT NULL,
  symbol      TEXT,                -- function/struct name (optional)
  link_type   TEXT NOT NULL,       -- implements | tests | documents
  created_at  TEXT NOT NULL,
  PRIMARY KEY (spec_id, file_path, link_type)
)

history (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  spec_id     TEXT NOT NULL REFERENCES specs(id),
  changed_at  TEXT NOT NULL,
  field       TEXT NOT NULL,
  old_value   TEXT,
  new_value   TEXT
)
```

### ID Generation

`bsg-` prefix + 4 hex chars from hash of name + timestamp. Check for collisions, extend to 6 chars if needed.

## CLI Commands

### Core CRUD

```
bsg init                                     # Create .bsg/bsg.db in current directory
bsg add "Weight Entry Validation" \
  --type behavior \
  --tag weight,input \
  --body "Entries must be 50-1000 lbs..."    # Create spec
bsg add "Weight Entry Validation"            # No --body opens $EDITOR
bsg show bsg-a3f2                            # Full detail + edges + code links + history
bsg edit bsg-a3f2                            # Opens $EDITOR with current spec
bsg edit bsg-a3f2 --name "New Name"          # Inline field edit
bsg edit bsg-a3f2 --body "new body"
bsg edit bsg-a3f2 --tag weight,validation
bsg rm bsg-a3f2                              # Delete spec + edges + links
bsg rm bsg-a3f2 --force                      # Skip confirmation
```

### Status Transitions

```
bsg accept bsg-a3f2
bsg implement bsg-a3f2
bsg verify bsg-a3f2
bsg deprecate bsg-a3f2 --superseded-by bsg-ff10
```

State machine:
```
draft → accepted → implemented → verified
                                    ↓
                               deprecated
```
Any status can also go directly to deprecated. No backward transitions. Each transition recorded in history.

### Relationships

```
bsg link bsg-a3f2 --depends-on bsg-b1c4
bsg link bsg-a3f2 --refines bsg-0012
bsg link bsg-a3f2 --conflicts-with bsg-ee31
bsg link bsg-a3f2 --supersedes bsg-dd44
bsg unlink bsg-a3f2 bsg-b1c4                # Remove all edges between two specs
bsg unlink bsg-a3f2 bsg-b1c4 --relation depends_on
```

Cycle detection on `depends_on` edges (DAG enforcement via DFS).

### Code Traceability

```
bsg trace bsg-a3f2 --file src/weight.go:Validate
bsg trace bsg-a3f2 --file src/weight_test.go --as tests
bsg untrace bsg-a3f2 src/weight.go
bsg trace src/weight.go                     # Reverse: which specs cover this file?
```

### Querying

```
bsg list                                    # All specs, grouped by status
bsg list --status draft
bsg list --type constraint
bsg list --tag payment
bsg list --unlinked                         # Orphan specs (no edges)
bsg list --blocking                         # Specs that block others
bsg list --no-code                          # Specs with no code links
```

### Graph

```
bsg graph                                   # ASCII dependency tree
bsg graph --dot                             # Graphviz DOT output
bsg graph --from bsg-a3f2                   # Subgraph from one spec
bsg path bsg-a3f2 bsg-ff10                  # Shortest path between two specs
```

### Analysis

```
bsg coverage                                # Summary: N specs, M with code links, K verified
bsg drift                                   # Specs whose linked files changed since last verify
```

`bsg drift` compares file mtimes against the `updated_at` of the most recent `verify` status change in history.

### Export

```
bsg export --json                           # Full dump
bsg export --jsonl                          # One spec per line, git-friendly
bsg export --markdown                       # Readable doc grouped by status/type
```

## Project Structure

```
/Users/jasonmay/jml/bsg/
  go.mod
  go.sum
  main.go                    # Entry point
  cmd/
    root.go                  # Root cobra command
    init.go                  # bsg init
    add.go                   # bsg add
    show.go                  # bsg show
    edit.go                  # bsg edit
    rm.go                    # bsg rm
    status.go                # accept/implement/verify/deprecate
    link.go                  # bsg link / unlink
    trace.go                 # bsg trace / untrace
    list.go                  # bsg list
    graph.go                 # bsg graph / path
    coverage.go              # bsg coverage / drift
    export.go                # bsg export
    prime.go                 # bsg prime
    check_file.go            # bsg check-file
    setup.go                 # bsg setup claude
  internal/
    db/
      db.go                  # Open, migrate, connection management
      schema.go              # SQL schema + migrations
      specs.go               # Spec CRUD queries
      edges.go               # Edge queries + cycle detection
      links.go               # Code link queries
      history.go             # History append + query
    model/
      spec.go                # Spec struct + validation
      edge.go                # Edge struct + relation types
      link.go                # CodeLink struct
      history.go             # HistoryEntry struct
    id/
      id.go                  # bsg-XXXX generation + collision handling
    graph/
      graph.go               # DAG ops: topo sort, path finding, cycle detection
      dot.go                 # Graphviz DOT export
      ascii.go               # ASCII tree rendering
    editor/
      editor.go              # $EDITOR integration for add/edit
    display/
      table.go               # Table formatting for list output
      detail.go              # Detail view for show
```

## Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| CLI framework | cobra | Standard for Go CLIs, subcommand ergonomics justify the dep |
| SQLite driver | modernc.org/sqlite | Pure Go, no CGo, simple cross-compilation |
| DB location | `.bsg/bsg.db` in project root | Walk up dirs to find it (like `.git/`). Fail fast if missing. |
| ID format | `bsg-` + 4 hex | Short, readable, collision-checked |
| Cycle detection | DFS on `depends_on` only | Other relation types are unconstrained |
| History | Append-only | Every mutation logged, never modified |
| Editor | `$EDITOR` with template | Parse after close, abort if unchanged |

## Dependencies

```
github.com/spf13/cobra      # CLI framework
modernc.org/sqlite           # Pure Go SQLite
```

Two deps total.

## Implementation Order

1. `go.mod` + `main.go` + `cmd/root.go` — skeleton
2. `internal/model/` — all structs and validation
3. `internal/id/` — ID generation
4. `internal/db/` — schema, connection, migrations
5. `internal/db/specs.go` — spec CRUD
6. `cmd/init.go` — `bsg init`
7. `cmd/add.go` + `internal/editor/` — `bsg add`
8. `cmd/show.go` + `internal/display/` — `bsg show`
9. `cmd/edit.go` — `bsg edit`
10. `cmd/rm.go` — `bsg rm`
11. `cmd/status.go` — accept/implement/verify/deprecate
12. `internal/db/edges.go` + `internal/graph/` — edges + cycle detection
13. `cmd/link.go` — `bsg link` / `unlink`
14. `internal/db/links.go` + `cmd/trace.go` — code traceability
14a. `cmd/prime.go` — `bsg prime` (depends on list, coverage, drift logic)
14b. `cmd/check_file.go` — `bsg check-file` (depends on code links)
14c. `cmd/setup.go` — `bsg setup claude` (standalone, writes JSON)
15. `cmd/list.go` — querying with filters
16. `cmd/graph.go` + `internal/graph/dot.go` + `ascii.go` — visualization
17. `cmd/coverage.go` — coverage + drift
18. `cmd/export.go` — JSON/JSONL/markdown
19. `internal/db/history.go` — wire history into all mutations
20. Tests throughout (table-driven, -race)

## Drift Detection (detailed)

### Concept

Drift = code changed but spec hasn't been re-verified. A spec is "drifting" when any of its linked files have been modified after the spec was last verified. This catches specs that claim to be verified but whose underlying implementation has moved on.

### Algorithm

```
for each spec with status = "verified":
    last_verify_time = latest history row where spec_id = spec.id AND field = "status" AND new_value = "verified"
    for each code_link where spec_id = spec.id:
        file_mtime = stat(code_link.file_path).ModTime()
        if file_mtime > last_verify_time:
            mark spec as drifted
            record which file(s) triggered it
```

### CLI Output

```
bsg drift
```

```
DRIFTED (3 specs)

  bsg-a3f2  Weight Entry Validation    verified 2025-12-01
            src/weight.go              modified 2025-12-15  (+14d)
            src/weight_test.go         modified 2025-12-10  (+9d)

  bsg-b1c4  BMI Calculation            verified 2025-11-20
            src/bmi.go                 modified 2025-12-18  (+28d)

  bsg-0012  Export CSV Format          verified 2025-10-05
            src/export.go              modified 2025-12-01  (+57d)

CLEAN (12 specs verified, no file changes)
```

### Flags

```
bsg drift                     # All drifted specs
bsg drift --days 7            # Only flag drift older than 7 days (ignore recent edits)
bsg drift --file src/weight.go  # Which specs drift because of this file?
bsg drift --json              # Machine-readable output
```

### Edge Cases

| Case | Behavior |
|---|---|
| Linked file deleted | Report as `MISSING` — file gone, spec can't be verified |
| Linked file is in `.gitignore` | Still tracked — bsg doesn't care about git, only file existence |
| Spec never verified | Not included — drift only applies to `verified` status |
| Spec verified multiple times | Use the most recent verify timestamp |
| Symlinked file | Resolve symlink, use real file's mtime |
| File touched but content unchanged | Still flagged — mtime is the signal, not content hash (keep it simple, v1) |

### Future (v2, not in scope)

- Content hash comparison instead of mtime (more accurate, slower)
- Git-aware drift: `git log --since=<verify_time> -- <file>` to check actual commits, not just mtime
- Auto-transition: `bsg drift --demote` to move drifted specs back to `implemented`

---

## Claude Code Integration (detailed)

### Concept

BSG integrates with Claude Code via its hooks system. Rather than relying on manual CLAUDE.md instructions, BSG installs hooks that automatically inject spec context at the right moments: session start, file edits, compaction, and session end. BSG provides structured output that hooks consume silently.

### `bsg prime` command

Outputs a concise context block (~1-2k tokens) to stdout. Designed to be consumed by Claude Code hooks.

```
# BSG Spec Context
## Coverage: 12 specs total, 8 with code links, 5 verified
## Drifted (2):
  bsg-a3f2 "Weight Entry Validation" — src/weight.go modified +14d since verify
  bsg-b1c4 "BMI Calculation" — src/bmi.go modified +28d since verify
## Ready to implement (3):
  bsg-0012 "Export CSV Format" [accepted, no code links]
  bsg-ee31 "Input Sanitization" [accepted, no code links]
  bsg-ff10 "Dark Mode Support" [accepted, no code links]
```

Flags:
- `--json` for machine-readable output
- `--compact` for minimal output (just IDs and counts, for low-context situations)

### `bsg check-file <file_path>` command

Given a file path (from PostToolUse hook stdin), outputs which specs are linked to that file and their status. Used by the PostToolUse hook to provide real-time spec awareness when editing files.

```
bsg check-file src/weight.go
# Linked specs:
#   bsg-a3f2 "Weight Entry Validation" [verified] — implements:Validate
#   bsg-c1d3 "Weight Range Constraint" [accepted] — implements
```

Returns exit 0 with empty stdout if no specs are linked (hook is silent).

### `bsg setup claude` command

Auto-installs hooks into `.claude/settings.json` (project-level). Installs:

1. **SessionStart hook** — runs `bsg prime` on session start, injects spec context
2. **PreCompact hook** — runs `bsg prime --compact` before compaction
3. **PostToolUse hook** (matcher: `Edit|Write`) — extracts `file_path` from stdin JSON, runs `bsg check-file <path>`, surfaces linked specs
4. **Stop hook** — runs `bsg prime --compact` as end-of-session drift reminder

Generated hooks config:
```json
{
  "hooks": {
    "SessionStart": [{
      "hooks": [{
        "type": "command",
        "command": "bsg prime 2>/dev/null || true"
      }]
    }],
    "PreCompact": [{
      "hooks": [{
        "type": "command",
        "command": "bsg prime --compact 2>/dev/null || true"
      }]
    }],
    "PostToolUse": [{
      "matcher": "Edit|Write",
      "hooks": [{
        "type": "command",
        "command": "jq -r '.tool_input.file_path // empty' | xargs -I{} bsg check-file {} 2>/dev/null || true"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "bsg prime --compact 2>/dev/null || true"
      }]
    }]
  }
}
```

`bsg setup claude` merges into existing settings (doesn't overwrite). `bsg setup claude --remove` removes BSG hooks.

### Machine-Readable Output

Every query command supports `--json` for agent consumption:

```
bsg show bsg-a3f2 --json
bsg list --status accepted --json
bsg drift --json
bsg coverage --json
bsg graph --from bsg-a3f2 --json
```

JSON output schema for `bsg show --json`:
```json
{
  "id": "bsg-a3f2",
  "name": "Weight Entry Validation",
  "type": "behavior",
  "status": "accepted",
  "body": "Entries must be 50-1000 lbs...",
  "tags": ["weight", "input"],
  "edges": [
    {"to": "bsg-b1c4", "relation": "depends_on"},
    {"from": "bsg-0012", "relation": "refines"}
  ],
  "code_links": [
    {"file": "src/weight.go", "symbol": "Validate", "type": "implements"}
  ],
  "created_at": "2025-12-01T10:00:00Z",
  "updated_at": "2025-12-15T14:30:00Z"
}
```

### Agent Workflows

**1. "What should I build next?"**
```bash
bsg list --status accepted --no-code --json
```

**2. "What does this spec require?"**
```bash
bsg show bsg-a3f2 --json
```

**3. "I just wrote code — what specs does it satisfy?"**
```bash
bsg trace src/weight.go
```

**4. "Is anything broken?"**
```bash
bsg drift --json
```

### What BSG Does NOT Do

| Not in scope | Why |
|---|---|
| Embed an LLM or agent runtime | BSG is a data layer, not an orchestrator |
| Auto-generate specs from code | Specs are human-authored intent; code is implementation |
| Talk to APIs (GitHub, Linear, etc.) | Keep it local-first; export formats enable integration |
| Watch files or run as a daemon | CLI tool, not a service. `bsg drift` is run on demand. |

### Key Design Decision

Hooks fail silently (`2>/dev/null || true`) so BSG never blocks Claude Code. If the DB doesn't exist or a command fails, the hook produces no output and exits 0.

### Future (v2, not in scope)

- `bsg mcp` — expose BSG as an MCP server so agents can query specs via tool use without shell
- `bsg hook pre-commit` — git hook that warns if modified files are linked to verified specs (nudge to re-verify)
- `bsg suggest` — given a file path, suggest which unlinked specs might be relevant (fuzzy match on names/tags vs file path/symbols)

---

## Verification

1. `bsg init` creates `.bsg/bsg.db` with correct schema
2. `bsg add "Test Spec" --type behavior --body "must do X"` creates spec with `bsg-XXXX` id
3. `bsg show bsg-XXXX` displays full detail
4. `bsg link bsg-XXXX --depends-on bsg-YYYY` creates edge; circular dep is rejected
5. `bsg trace bsg-XXXX --file main.go:Foo` links code
6. `bsg list --status draft` filters correctly
7. `bsg accept bsg-XXXX` transitions status, recorded in history
8. `bsg drift` detects when linked files changed after verification
9. `bsg graph --dot | dot -Tpng -o graph.png` produces valid graphviz
10. `bsg export --json | jq .` produces valid JSON
