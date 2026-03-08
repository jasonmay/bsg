# BSG — Behavioral Spec Graph

BSG is an LLM-first spec tracking tool for requirements, expectations, and intentions.
Specs live as JSON files in .bsg/specs/ and are version-controlled with git. The SQLite
database is a local cache rebuilt automatically from these files.

## IDs

Every spec gets an auto-generated ID like `bsg-a3f2`. The `bsg add` command prints the
generated ID to stdout. All other commands use this ID, not the spec name.

## Commands

| Command | Description |
|---------|-------------|
| bsg add <name> --type <type> [--body <text>] [--tag <csv>] | Create a spec, prints generated ID. Reads body from stdin if piped. |
| bsg show <id> [--json] | Display a spec, its links, and relationships |
| bsg status <id> <new-status> | Transition spec status (e.g. draft -> accepted) |
| bsg delete <id> | Delete a spec and its links |
| bsg link <id> --depends-on <id> | Create a spec-to-spec relationship |
| bsg unlink <from> <to> [--relation <rel>] | Remove spec-to-spec relationships |
| bsg trace <id> --file <path> [--as type] | Link a spec to code (--as: implements, tests, documents) |
| bsg untrace <id> <file> | Remove a code link |
| bsg summarize | Print summary of all specs with relationships |
| bsg prime [--compact] [--json] | Show spec coverage and status |
| bsg inspect [path[:line[:col]]] [--recursive] | Show specs for a file, directory, or cwd |
| bsg sync | Rebuild database from spec files |
| bsg setup claude [--dry-run] [--remove] | Install/remove Claude Code hooks |
| bsg tags [prefix] [--json] | List tags with counts, or filter by prefix |
| bsg tree [path] | Show file tree of spec-linked files with colored status |

## Spec Types

| Type | When to use |
|------|-------------|
| behavior | What the system should do — user-visible actions, responses, side effects |
| constraint | Limits and rules — validation, rate limits, size bounds, permissions |
| interface | API contracts — endpoints, function signatures, protocols, data formats |
| data-shape | Data structures — schemas, models, field definitions, relationships |
| invariant | Things that must always be true — consistency rules, ordering guarantees |

## Spec Lifecycle

draft -> accepted -> implemented -> verified -> deprecated -> archived

Any pre-verification status (draft, accepted, implemented) can transition to paused.
Paused specs can resume to draft, accepted, or implemented.
Any status can also transition directly to archived.

## Trace syntax

`--file` accepts: `file`, `file:Symbol`, `file:10-25` (line range), `file:10:5-25:0` (line:col range)

## Spec Relationships

Specs can be linked with directed edges via bsg link:

  bsg link bsg-X --depends-on bsg-Y
  bsg link bsg-X --refines bsg-Y
  bsg link bsg-X --conflicts-with bsg-Y
  bsg link bsg-X --implements bsg-Y
  bsg link bsg-X --supersedes bsg-Y

depends_on edges are cycle-checked. Edges appear in both specs' JSON files.
Remove with bsg unlink bsg-X bsg-Y [--relation <rel>].
Use bsg why <file> --recursive to traverse edges and show upstream specs.

## Worked Example

```
$ bsg add "Weight entries must be positive" --type constraint --body "Reject zero or negative weight values at input" --tag validation,weight
bsg-7f1a

$ bsg trace bsg-7f1a --file src/weight.go:ValidateWeight
traced bsg-7f1a -> src/weight.go:ValidateWeight (implements)

$ bsg trace bsg-7f1a --file src/weight_test.go --as tests
traced bsg-7f1a -> src/weight_test.go (tests)

$ bsg status bsg-7f1a accepted
bsg-7f1a -> accepted

$ bsg status bsg-7f1a implemented
bsg-7f1a -> implemented

$ bsg show bsg-7f1a
ID:         bsg-7f1a
Name:       Weight entries must be positive
Type:       constraint
Status:     implemented
...

$ bsg inspect src/weight.go
File:
  bsg-7f1a "Weight entries must be positive" [implemented] (implements) :ValidateWeight

$ bsg inspect src/ --recursive
(shows all specs under src/ + upstream specs via edges)

$ bsg delete bsg-7f1a
deleted bsg-7f1a
```

## File Format

Spec files in .bsg/specs/<id>.json contain: id, name, type, status, body, tags, links, and edges.
These files are the source of truth — commit them to git.
