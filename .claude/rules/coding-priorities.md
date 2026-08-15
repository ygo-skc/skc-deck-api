---
paths:
  - "**/*.go"
---

# Coding priorities (read before writing or changing any Go)

When adding or modifying code in this repo, optimize in this order: **correctness → idiomatic Go →
performance → low memory footprint**. This is a small (~1.3k LOC) HTTPS API on port 9010 that stores
and serves deck lists out of MongoDB and hydrates them through one gRPC call to ygo-service — the
per-request path is short, so the last two priorities are about *not regressing* the reuse and
projection patterns already in place, not about adding machinery.

**Never hand-edit generated code.** `deck/` is protoc output from `deck.proto` (regenerate with
`make generate-grpc`). It is gitignored, and no Go file currently imports it.

**Idiomatic Go**
- Follow Effective Go / Go Code Review Comments: short names in small scopes, `err != nil` handled
  immediately, no needless getters, accept interfaces & return concrete types, keep interfaces small
  and defined at the consumer (as `SKCDeckAPIDAO` is in `db/skc_deck_db.go`).
- Match the surrounding code: package-level dependency vars, not framework DI — the DAO is wired
  once at `api/server.go:32` and tests reassign it.
- Logging is request-scoped. A handler opens the request with
  `cUtil.InitRequest(req.Context(), apiName, <op>, attrs...)` — where `<op>` is a named const at the
  top of the handler file — enriches it with `cUtil.AddLoggerAttribute`, and passes the returned ctx
  down; everything below pulls it back with `cUtil.RetrieveLogger(ctx)`. Bare `slog.` is correct
  **only** where there is no request ctx: startup (`main.go`, `db/connect.go`, `api/server.go`) and
  the globally-registered validator callbacks in `validation/`.
- Errors crossing a layer are `*cModel.APIError`, never a bare `error`. Handlers surface them and
  return immediately — `err.HandleServerResponse(res)` when you already hold one, or
  `cModel.HandleServerResponse(cModel.APIError{...}, res)` when constructing it inline.
- Compare sentinel errors with `errors.Is`, not by matching the message string — e.g.
  `errors.Is(err, mongo.ErrNoDocuments)` for a missing document.
- Request payloads are validated by struct tags on the model plus the custom validators registered
  in `validation/`; don't hand-roll field checks in handlers.
- Prefer the standard library and the shared `skc-go/common/v3` helpers over new dependencies. Use
  `gofmt` semantics and leave no vet warnings. `go test ./...` runs with coverage on every PR
  (`.github/workflows/unit-test.yaml`), but **`go vet` is not automated anywhere** — run it yourself.

**Performance**
- Reuse expensive objects rather than rebuilding them per request: the gzip writer `sync.Pool`
  (`api/server.go:35-40`) and package-level compiled regexes
  (`io/deck_list_serialization.go:18`, `validation/`). Never compile a regex inside a request path.
- Batch downstream calls. A whole deck is hydrated with a single `GetCardsByIDProto`
  (`io/deck_list_serialization.go:30`); never call ygo-service per card in a loop.
- Let Mongo do the work: filter in the query and `SetProjection` down to the fields actually used
  (`db/skc_deck_db.go:111-116`) so unused fields never cross the wire.
- Keep queries index-covered. `createIndexes()` in `db/connect.go` is the inventory and this repo
  **owns** it — a new filter or sort means checking coverage and adding an index there. Remember the
  compound-prefix rule: `{uniqueCards:1, createdAt:-1}` serves a query leading with `uniqueCards`,
  not one filtering on `createdAt` alone.
- Do work once: hoist invariants out of loops and bind a repeated map lookup to a local instead of
  indexing the same key several times (see `DeckListBreakdown.Partition`, `model/deck_list.go:55-63`).
- **There is no concurrency in this repo today, and that is correct** — parsing gates the single
  downstream fetch, so the flow is genuinely sequential. Reach for `cUtil.AtomicWaitGroup[T]` or
  `sync.WaitGroup` only if independent work actually appears; don't fan out dependent calls.

**Low memory footprint**
- Preallocate slices and maps whose size is predictable from the input —
  `make([]Content, 0, len(dlb.MainDeck))` (`model/deck_list.go:74`) is the pattern to follow, and
  `len(tokens)` is available where `io/deck_list_serialization.go:49-50` builds its map and slice.
  Don't preallocate when the size genuinely isn't known.
- Build strings with `strings.Builder` (`model/deck_list.go:87-101`), not `+=` in a loop.
- Copying is mostly cheap here — `cModel.YGOCard` is an interface, so ranging a card list copies an
  interface header, not a struct. Don't invent copy-avoidance findings. Do range by index when the
  element is a genuinely large **value** struct, as `[]model.DeckList` is.
- Don't hold whole result sets longer than needed; project early (above) rather than decoding fields
  you discard.
- Prefer streaming/in-place transforms over building throwaway intermediate slices.

Don't add speculative micro-optimizations that hurt readability, and don't add defensive guards for
failure modes that can't occur in this environment — a `strconv.Atoi` whose input a regex already
guaranteed needs no error branch. Keep changes idiomatic and measured.
