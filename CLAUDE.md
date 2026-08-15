# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go HTTP/2 (TLS) API for storing and retrieving Yu-Gi-Oh! deck lists — deck submission, deck
retrieval, and "which decks feature this card". Built with [chi](https://github.com/go-chi/chi),
backed by MongoDB, and dependent on `ygo-service` (gRPC, via the shared `skc-go/common/v3` client)
for card data. All routes live under `/api/v1/deck` on port **9010**.

See `README.md` for the feature list and **`SYSTEM_DESIGN.md` for per-endpoint flows, the data model,
and the deck-list encoding — it is kept current, so consult it before changing handler flows.**

## Coding priorities

Optimize in this order: **correctness → idiomatic Go → performance → low memory footprint**. Full
checklist (idiomatic Go / performance / memory): `.claude/rules/coding-priorities.md`, which loads
automatically whenever you touch a `.go` file.

**Never hand-edit generated code.** `deck/` is protoc output from `deck.proto` (`make generate-grpc`).
It is gitignored, and no Go file currently imports it.

## Commands

| Command | Purpose |
| --- | --- |
| `go run .` | Run locally — serves **HTTPS on port 9010** (needs certs and `SKC_DECK_API_DOT_ENV_FILE`) |
| `make test` / `go clean -testcache && go test ./...` | Run all tests (cache cleared) |
| `go test ./api -run TestName` | Run a single test |
| `make coverage` | Tests with coverage, opens HTML report |
| `make build` | `go mod tidy` + cross-compile Linux ARM64 static binary |
| `go vet ./...` | Vet — **not automated anywhere**, so run it yourself |
| `make generate-grpc` | Regenerate `deck/` from `deck.proto` |

CI (`.github/workflows`) runs unit tests with coverage on every PR, plus CodeQL. Note `make build`
does **not** vet, and there is no lint step beyond `go vet`. Renovate automerges `minor`, `patch`,
`pin`, and `digest` updates, so direct deps sit on the latest patch by construction.

## Architecture

Layered, with no framework-based DI — dependencies are package-level variables that tests reassign.

```
main.go → downstream.ConnectToYGOService() + db.EstablishSKCDeckAPIDBConn() + api.RunHttpServer()

api/          chi router, middleware, one file per handler. HTTPS-only: TLS 1.3 + HTTP/2, gzip via
              sync.Pool, API-Key middleware available for admin routes. Handlers open a request with
              cUtil.InitRequest and surface *cModel.APIError via HandleServerResponse.
db/           MongoDB DAO. SKCDeckAPIDAO interface + ...Implementation struct. Owns its index
              inventory in connect.go's createIndexes().
downstream/   ygo-service gRPC client, constructed from skc-go/common/v3/client.
io/           Deck-list (de)serialization: parse the base64 content into card IDs + quantities, then
              hydrate through one batch GetCardsByIDProto call.
model/        DeckList persistence/response structs and DeckListBreakdown (partition, sort, validate).
validation/   go-playground/validator wrappers plus custom deck-name and mascot validators.
```

Data lives in Mongo database `deckDB`, collection `lists` (X.509 auth). Deck contents are stored
base64-encoded in `content`; `uniqueCards`, `createdAt`, and `tags` are indexed, plus a compound
`{uniqueCards:1, createdAt:-1}`.

## Skills

- `go-drift-check` — periodic health audit (idiomatic Go, Mongo index coverage, performance and
  memory, adoptable dependency/stdlib additions).
