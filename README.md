# BSG — Behavioral Spec Graph

An LLM-first spec tracking tool for requirements, expectations, and intentions.

BSG tracks **what your system should do** — living specs that evolve, link to code, and can be verified. Unlike task trackers, BSG manages behavioral specifications as a dependency-aware graph with code traceability and drift detection.

Specs persist as JSON files in `.bsg/specs/` and are version-controlled with git. The SQLite database is a local query cache rebuilt automatically from these files.

## Install

```
go install github.com/jasonmay/bsg@latest
```

## Quick Start

```bash
# Initialize in your project
bsg init

# Create a spec
bsg add "User login validation" --type behavior \
  --body "Must reject empty passwords and usernames shorter than 3 chars" \
  --tag auth,validation
# prints: bsg-a3f2

# Link it to code
bsg trace bsg-a3f2 --file src/auth.go:ValidateLogin
bsg trace bsg-a3f2 --file src/auth_test.go --as tests

# Move through lifecycle
bsg status bsg-a3f2 accepted
bsg status bsg-a3f2 implemented
bsg status bsg-a3f2 verified

# Check coverage
bsg prime
```

## Commands

| Command | Description |
|---------|-------------|
| `bsg init` | Initialize `.bsg/` in current directory (idempotent) |
| `bsg add <name> --type <type>` | Create a spec, prints generated ID |
| `bsg show <id>` | Display a spec, its links, and history |
| `bsg status <id> <status>` | Transition spec status |
| `bsg delete <id>` | Delete a spec and its links |
| `bsg trace <id> --file <path>` | Link a spec to code |
| `bsg untrace <id> <file>` | Remove a code link |
| `bsg prime` | Show spec coverage, drifted specs, and what's ready |
| `bsg check-file <path>` | Show specs linked to a file |
| `bsg sync` | Force rebuild database from spec files |
| `bsg setup claude` | Install Claude Code hooks |
| `bsg lsp` | Start LSP server (stdio) |

### `bsg add`

```bash
bsg add "Spec name" --type behavior --body "Description" --tag auth,core
bsg add "Spec name" --type constraint                  # opens $EDITOR
echo "Long description..." | bsg add "Spec name" --type behavior  # reads from stdin
```

Flags: `--type` (required), `--body`, `--tag` (comma-separated)

### `bsg trace`

The `--file` flag accepts several formats:

```
--file src/main.go                 # whole file
--file src/main.go:Validate        # symbol
--file src/main.go:10-25           # line range
--file src/main.go:10:5-25:0       # line:col range
```

Link type via `--as`: `implements` (default), `tests`, `documents`

## Spec Types

| Type | When to use |
|------|-------------|
| `behavior` | What the system should do — user-visible actions, responses, side effects |
| `constraint` | Limits and rules — validation, rate limits, size bounds, permissions |
| `interface` | API contracts — endpoints, function signatures, protocols, data formats |
| `data-shape` | Data structures — schemas, models, field definitions, relationships |
| `invariant` | Things that must always be true — consistency rules, ordering guarantees |

## Spec Lifecycle

```
draft → accepted → implemented → verified → deprecated → archived
```

Any status can also transition directly to `archived`.

## Drift Detection

`bsg prime` detects **drifted specs** — specs marked as `verified` whose linked files have been modified since verification. This catches specs that claim to be verified but whose implementation has moved on.

```
# BSG Spec Context
## Coverage: 12 specs total, 8 with code links, 5 verified
## Drifted (2):
  bsg-a3f2 "Weight Entry Validation" — src/weight.go modified +14d since verify
  bsg-b1c4 "BMI Calculation" — src/bmi.go modified +28d since verify
## Ready to implement (3):
  bsg-0012 "Export CSV Format" [accepted, no code links]
```

## Claude Code Integration

BSG is designed to work with coding agents. Run `bsg setup claude` to install hooks that automatically:

- **Session start**: inject spec context via `bsg prime`
- **File edits**: surface linked specs via `bsg check-file`
- **Compaction/stop**: refresh spec context via `bsg prime --compact`

The setup also adds a reference to `.bsg/README.md` in your `CLAUDE.md`, so the agent discovers BSG commands on its own.

Hooks fail silently (`2>/dev/null || true`) so BSG never blocks your agent.

## Git-Portable Specs

Specs live as JSON files in `.bsg/specs/<id>.json`:

```json
{
  "id": "bsg-a3f2",
  "name": "Weight Entry Validation",
  "type": "behavior",
  "status": "verified",
  "body": "Weight entries must be positive numbers...",
  "tags": ["weight", "input"],
  "links": [
    {"file": "src/weight.go", "symbol": "Validate", "type": "implements"}
  ]
}
```

The SQLite database (`.bsg/bsg.db`) is gitignored — it's rebuilt automatically from spec files when needed. After a `git clone`, the first `bsg` command recreates the database.

Force rebuild anytime with `bsg sync`.

## LSP

`bsg lsp` provides an LSP server (stdio transport) with:

- **Hover**: show spec info when hovering linked code ranges
- **Diagnostics**: highlight code ranges linked to specs
- **Go to definition**: jump from `bsg-XXXX` comments to the spec

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Storage | SQLite + JSON files | SQLite for fast queries, JSON for git portability |
| SQLite driver | `modernc.org/sqlite` | Pure Go, no CGo, simple cross-compilation |
| CLI framework | `cobra` | Standard for Go CLIs |
| DB location | `.bsg/bsg.db` | Walk-up-dirs discovery like `.git/` |
| ID format | `bsg-` + 4 hex | Short, readable, collision-checked |
| History | Append-only, DB-only | Git log covers audit trail for spec files |

## Acknowledgments

Inspired by [Beads](https://github.com/steveyegge/beads) by Steve Yegge, which demonstrated that coding agents need structured memory systems to work effectively. BSG takes a complementary approach — where Beads tracks tasks and agent memory, BSG tracks behavioral specifications and their relationship to code.
