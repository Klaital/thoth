# Design Doc for Thoth's Collection website

Thoth's Collection is a website for organizing your collections. The specific products in mind for the MVP are Magic: the Gathering cards and sealed product. Ideally features will be added later to manage other collections, such as books and DVDs.

## General notes

Backend written in Go

Frontend as React SPA, written in Typescript.

To be deployed as a subdomain on a VPS somewhere.

Use generated code wherever practical
 - The backend should manage the database code using sqlx
 - The backend API should be declared using Protobuf.
	- Because we need to support a webapp, use gRPC-Gateway as a proxy.
	- Use the Protobuf files to generate the frontend's client library

Use formatting linters to ensure uniform code styles.

# Frontend

React SPA in Typescript.

Accounts can be created using direct email+password, or via OAuth providers.
	- Out of the box, only Google auth is supported.

Use bookmarkable URLs wherever reasonable.

Are there automatic linters like `go fmt` for Typescript?

# Backend

Database will be a mysql database hosted over the internet.

Use `go fmt` to format the code.

Data for the cards will be periodically loaded from Scryfall's bulk data APIs. (we'll need a client library for that, too)

## Observability

OpenTelemetry, exported over OTLP/HTTP to Grafana Cloud (free tier). Off unless
`OTEL_EXPORTER_OTLP_ENDPOINT` is set (see `.env.example`); `internal/telemetry`
does the setup for both `thoth-api` and `thoth-sync`.

- **RPC metrics and traces:** `otelgrpc` on the gRPC server plus `otelhttp` on the
  HTTP mux. Each request traces HTTP → gRPC → SQL (`otelsql` in `db.Connect`).
  Spans carry `enduser.id`.
- **DB:** query latency and connection-pool stats. Only queries inside a request
  are traced, so migrations and the sync job don't flood Tempo.
- **Usage:** `thoth.*` counters (logins, registrations, collections/decks created,
  cards added, bulk-import lines, deck checkouts/returns, sync runs) and
  `thoth.*.total` gauges, defined in `internal/telemetry/metrics.go`. Rule:
  metric attributes take small fixed value sets only. User IDs go on spans,
  never on metrics.
- **Logs:** `slog` (and so `log.Printf`) goes to stdout and OTLP, with the
  trace ID attached.
- **Frontend:** Grafana Faro (`frontend/src/telemetry.ts`) for sessions, views
  (numeric IDs stripped), Web Vitals, JS errors, and `traceparent` on API
  calls. It's compiled in only when `VITE_FARO_URL` is set at build time (the
  CI repo variable).
- **Local:** `docker compose --profile observability up -d` runs `grafana/otel-lgtm`
  on http://localhost:3000.
- **Dashboards:** `deploy/grafana/*.json` (API health, Usage) are the source of
  truth. `deploy/grafana/push.sh` upserts them into a "Thoth" folder. CI runs it
  after each prod deploy when the `GRAFANA_URL` variable and the
  `GRAFANA_TOKEN` secret are set. UI edits get overwritten, so export the JSON
  and commit it.
  - Panels filter only on `job` (service name), the one label that's the same
    in Grafana Cloud and local Prometheus.
  - Rate panels use a 3m minimum interval because metrics are pushed every 60s.
  - Counters are created at 0 on startup (`telemetry.PrimeCounters`), so
    `increase()` doesn't miss the first event after each deploy.

## Data Model

1. Users
2. Collection
	- A physical grouping of things you own, like a bulk box or storage container.
	- Name, description. Freetext fields for users to remember which collection is which and where it's kept.
	- Type. Each collection holds one kind of item: MtG cards, sealed MtG product, Books or DVDs.
	- Foreign key to Users table to denote the Owner. 
	- Every user has exactly one default card collection (`is_default`), for users who don't want to manage separate groupings. It can't be deleted.
	- Collections and Items should only be viewable by the owner by default, unless the owner marks it as Public. It should always only be writable by the Owner.
3. Items
	- These are the child elements of a Collection. Different fields are relevant for different types of collections.
	- Cards (`collection_cards`) are exact printings: a foreign key into the Scryfall dataset, foil treatment, and the language of the physical card. Unique on (collection, printing, foil, language).
	- For Sealed Product, just a Set ID and Description may be sufficient. Ideally it should be enough to look up prices automatically somewhere. 
	- For Books and DVDs, a UPC field and Title.
	- Include a Notes field on all Items so users can write freetext notes like "borrowed this from Rich"
	- Include a Quantity field rather than allowing duplicates in the Collection.
	- Use separate tables for the different types of Items, but they should all use a forien key reference to their owning Collection.
4. Deck
	- A playable group of cards, owned by a user, separate from collections: a deck can be just an idea, holding cards you don't own.
	- Name, notes (where it's stored, how it plays, ideas to improve it), public flag.
	- Deck cards (`deck_cards`) are identified by card name ("City of Brass"), not printing. A preferred printing is optional. Each card sits on a board: commander, main or sideboard.
5. Deck checkouts
	- Physical cards pulled out of a collection to build a deck (`deck_checkouts`): deck, collection card row, quantity.
	- The collection card row never moves, so `collection_cards.quantity` is still the number owned, the copies in the box are quantity minus the checked-out total, and a return always goes back to the collection the card came from.
	- A deck with checked-out cards can't be deleted, and a collection card's quantity can't drop below its checked-out copies. Merging rows (move, language change) carries the checkouts to the surviving row.
	- Deck ownership is matched by card name: any printing, foil or language in the owner's card collections counts. Copies checked out to other decks don't.

Use Go classes for the application to handle these data objects separate from the sqlc models. Write helper functions to convert between database -> application -> wire protocol models.


