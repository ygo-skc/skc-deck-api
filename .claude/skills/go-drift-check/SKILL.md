---
name: go-drift-check
description: Use when the user asks to "audit the Go code", "check for idiomatic Go", "review Mongo indexes", "check for missing indexes", "check for unused indexes", "check for new library features we should adopt", "are we behind on any dependencies", "check for deprecated library usage", "check for performance or memory regressions", "run a drift check", "check for drift", or wants a health check of this repo's Go code. Also for periodic or pre-release code-quality audits of this repository, not for reviewing a single PR's diff.
---

# Go Drift Check

Audit this repo's Go code for drift across four dimensions and report findings — do not silently
fix anything unless the user asks. This is a reporting skill: read code, compare it against a
baseline, and list what has drifted with file:line references and a concrete fix suggestion.

## Scope

Audit the whole repo (`api/`, `db/`, `downstream/`, `io/`, `model/`, `validation/`, `main.go`).
**Exclude `deck/`** — protoc output from `deck.proto`, gitignored, and imported by no Go file. If the
user asks to check only recent work, scope to `git diff release...HEAD` instead.

`.claude/rules/coding-priorities.md` (linked from CLAUDE.md's Coding priorities section) is this
repo's standard. This skill is only the **audit procedure** for it — the rule says what good looks
like, the skill says how to find where the code drifted from it. **Re-read the rule at the start of
every run:** its `paths:` glob fires on Go *edits*, so a read-only audit cannot assume it is already
loaded, and it may be updated independently of this skill.

For a repo this size (~1.3k non-generated LOC), run checks 1–3 directly with `Bash`/`Grep`/`Read` —
they read overlapping files and produce findings that must be deduped against each other, and
second-hand `file:line` references can't be trusted without re-reading the code anyway.

**Exception:** dispatch check 4 (library capability drift) as a background subagent *before* starting
check 1, then collect its result when assembling the report. It's network-bound, reads only `go.mod`
plus release notes, and shares no findings with the other checks.

## 1. Idiomatic Go

The **"Idiomatic Go"** section of `.claude/rules/coding-priorities.md` is the authoritative checklist
here — re-read it before judging anything. What follows is how to audit against it, not a second
standard.

Mechanical baseline (single module, so plain `./...` works):
- `gofmt -l .` — anything listed is drift.
- `go vet ./...` — any warning is drift. Note **vet is not automated anywhere** in this repo (CI runs
  only `go test`, and `make build` skips vet), so this is the only thing catching vet regressions.

Targeted checks beyond that baseline:
- `grep -rn "slog\." api/ db/ io/ model/` for bare `slog` calls in request paths — the rule requires
  `cUtil.RetrieveLogger(ctx)` there. Bare `slog.` is legitimate only in `main.go`, `db/connect.go`,
  `api/server.go`, and `validation/` (validator callbacks are registered globally and have no ctx).
- Handler shape: each handler opens with `cUtil.InitRequest(req.Context(), apiName, <op>, ...)` where
  `<op>` is a named const at the top of the handler file. A handler that skips `InitRequest`, or
  inlines its operation name as a literal, is drift.
- Any new/changed method on `SKCDeckAPIDAO` (`db/skc_deck_db.go`) must exist on
  `SKCDeckAPIDAOImplementation`. There is **no mocks package in this repo** — don't go looking for
  one, and don't report its absence as drift.
- DB/downstream functions should return `*cModel.APIError`, not a bare `error`, at the boundary that
  feeds a handler; handlers surface it via `err.HandleServerResponse(res)` (or
  `cModel.HandleServerResponse(cModel.APIError{...}, res)` when constructing inline) and return
  immediately. `GetSKCDeckAPIDBVersion` returning a bare `error` is the one deliberate exception —
  the status handler only needs up/down.
- Sentinel errors compared by message string instead of `errors.Is` — e.g. anything matching
  `err.Error() ==`. Grep for it; the driver's message is not API.
- Per-card downstream loops: a deck is hydrated with a single `GetCardsByIDProto`
  (`io/deck_list_serialization.go:30`). A loop calling a per-ID method instead is drift.

## 2. Mongo index coverage (usage-driven, not a static list)

Unlike the sibling `ygo-service`, **this repo owns its index inventory**, so coverage is verifiable
here. Derive it fresh each run rather than trusting a hardcoded list:

1. Read `db/connect.go`'s `createIndexes()` (`db/connect.go:62`) to build the *current* inventory:
   for each index, its keys in order. Remember the compound-index prefix rule — an index on
   `{a:1, b:1}` serves queries filtering/sorting on `a` alone, or `a` then `b`; it does **not** serve
   a query filtering on `b` alone.
2. Find every operation against `deckListCollection` — grep `db/*.go` for `deckListCollection\.` —
   and read the filter (`bson.M{...}` / `bson.D{...}`) or `$match`/`$sort` fields actually used.
3. For each query, check whether an existing index covers the filtered/sorted field(s) as a usable
   prefix. `_id` lookups are always covered and need no action.
4. **Also check the inverse: indexes that no query uses.** This repo owns the inventory, so an index
   nothing queries still costs write throughput and memory on every insert. As of this writing only
   `cards_featured_in_deck` (`uniqueCards`) is exercised, by the `Find` at `db/skc_deck_db.go:118`;
   `decks_by_creation_date_desc`, `decks_by_tag`, and the compound
   `cards_featured_in_deck_by_creation_date_desc` have no query in this repo. Report that as an
   **observation, not a demand** — an index may exist for planned work, admin tooling, or an
   external consumer. Ask before suggesting a drop.
5. Weigh severity by how hot the path is, and don't demand an index for every field ever filtered —
   use judgment about query frequency and collection growth, consistent with the rule's instruction
   not to add work for scenarios that don't matter here.

Report each finding as: `<collection> is queried on {field(s)} by <DAO method> (db/....go:LINE) but
no current index covers that access pattern.`

## 3. Performance & memory

The **"Performance"** and **"Low memory footprint"** sections of `.claude/rules/coding-priorities.md`
are the checklist — apply them, don't re-derive new rules. Concretely, look for:

- Slices/maps built in a loop whose final size is already known but declared without a capacity hint.
  `make([]Content, 0, len(dlb.MainDeck))` (`model/deck_list.go:74`) is the pattern; the map and slice
  at `io/deck_list_serialization.go:49-50` have `len(tokens)` available and don't use it.
- Repeated map lookups of the same key that could bind to a local once — see
  `DeckListBreakdown.Partition` (`model/deck_list.go:55-63`), which indexes `dlb.AllCards[cardID]`
  several times per iteration.
- Regexes compiled inside a request path instead of at package level
  (`io/deck_list_serialization.go:18`, `validation/configure_validation.go:18-19`).
- New allocation-heavy hot-path code that doesn't reuse the gzip `sync.Pool` pattern
  (`api/server.go:35-40`).
- Mongo queries that don't `SetProjection` down to the fields actually used
  (`db/skc_deck_db.go:111-116` is the example to follow), letting unused fields cross the wire.
- Range-copy of large **value** structs — `cModel.YGOCard` is an interface, so card lists are cheap;
  `[]model.DeckList` is the one genuinely large value-struct slice. Don't manufacture findings here.
- **Don't flag sequential downstream calls as a concurrency bug.** Parsing gates the single batch
  fetch, so the current flow is correctly sequential. Only flag fan-out that is genuinely
  independent and written serially.

## 4. Library capability drift (new features worth adopting)

Per Scope, this should already be running in a background subagent — what follows is that agent's
brief. Run it inline only if no agent was dispatched.

**This is not a version-lag check** for most deps. `.github/renovate.json` automerges `minor`,
`patch`, `pin`, and `digest`, so those sit on the latest by construction — reporting "x.y.z →
x.y.z+1" is noise. The drift is the opposite: **new capabilities arrive and are never adopted.**

**One dependency worth checking directly every run:** `github.com/ygo-skc/skc-go/common/v3` is
first-party, released from the sibling `skc-go` monorepo on its own cadence with path-prefixed tags
(`common/vX.Y.Z`). The Go proxy resolves it normally and Renovate does pick it up, so a small gap is
usually just release timing rather than a defect — but it's cheap to confirm, and it matters when
someone has just cut a `common` release. Compare `go.mod` against the newest tag:
`git ls-remote --tags https://github.com/ygo-skc/skc-go "common/*"`. Report a gap only with its age;
a few days old is noise, a few minor versions behind is a finding.

Scope everything else to **releases from roughly the last 12 months**. Method:
1. `go list -m -u all` — one command, used only to spot a **major** version available. Don't build a
   report section out of its minor/patch output.
2. For each direct dependency whose API this repo actually exercises — `mongo-driver/v2`, `chi`,
   `go-playground/validator`, `cors`, `grpc`, `protobuf` — `WebFetch`/`WebSearch` its release notes
   for that window and look for additions that map onto something this repo does by hand. Grep the
   module path first so you know which symbols are in use. Skip plumbing this repo barely touches
   directly (`locales`, `universal-translator`, `x/net`, `uuid`).
3. Same for the **Go stdlib** (`go.mod` pins `go 1.26`) — additions that would replace hand-rolled
   code here. This repo already adopts modern stdlib (`strings.SplitSeq` at `api/server.go:109`), so
   the bar is real.
4. `skc-go/common/v3` — check its GitHub **release notes** for new shared helpers worth adopting.
   Its source lives in the sibling `skc-go` repo; release notes are enough here.

**Major version bumps deserve real time.** Renovate does *not* automerge majors, so one may be
sitting in an open PR; `gh pr list` is worth a look. Report what changed, which files import the
module, and roughly what migration would involve.

Deprecations count when they affect a symbol this repo imports, but they're secondary — lead with
adoptable additions. Every finding must tie back to concrete code: "mongo-driver added X" is not a
finding; "mongo-driver added X, which would replace the hand-rolled Y at `db/foo.go:120`" is. A
short, empty section 4 is the expected result most runs.

## What NOT to flag

- **Generated code in `deck/`.** See Scope.
- **The sequential parse-then-fetch flow.** See section 3.
- **Whatever the rule's closing line rules out** — speculative micro-optimizations that hurt
  readability, and defensive guards for failure modes that can't occur, such as the `strconv.Atoi`
  in `io/deck_list_serialization.go` whose input the regex already guarantees is `1`–`3`.
- **The absence of DAO mocks**, or of tests for packages that have none.

This skill finds drift from the standard `.claude/rules/coding-priorities.md` already states — it is
not license to invent stricter rules. When in doubt, quote the rule line the finding is drifting
from, plus the file:line where the code diverges.

## Report format

Respond directly in the conversation (no file needed unless the user asks). Structure:

```
## Go Drift Check

### 1. Idiomatic Go
- <file>:<line> — <what drifted> — <fix>
(or: "No drift found.")

### 2. Mongo index coverage
- <collection> queried on {field(s)} by <method> (<file>:<line>) — no covering index — <suggested index>
- OBSERVATION: index <name> has no query in this repo — <cost> — confirm it's still wanted
(or: "No drift found.")

### 3. Performance & memory
- <file>:<line> — <what drifted> — <fix>
(or: "No drift found.")

### 4. Library capability drift
- common/v3 in go.mod is <X> but newest tag is <Y>, released <N> ago
- <module/stdlib> added <feature> in <version> — could replace <what> at <file>:<line>
- MAJOR AVAILABLE: <module> <current> → <new major> — <what changed> — <files importing it>
(or: "No adoptable additions found in the last ~12 months.")
```

If the user then asks to fix a finding, treat it as a normal edit and rely on the repo's
`PostToolUse` hook (`.claude/settings.json` — gofmt + vet + test on the changed package) rather than
re-running the whole audit.
