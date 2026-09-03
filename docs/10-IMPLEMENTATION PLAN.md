---
title: Implementation Plan
document: 10-IMPLEMENTATION-PLAN
version: 2.0
status: Current
---

# Implementation Plan

## Purpose

This document is a technical map for contributors: the actual stack, the actual repository structure, the actual architectural patterns each layer follows, and how to build/run/test the application locally.

It deliberately does **not** duplicate what other documents already own:

- **Why Palladium exists and what it should feel like to operate** — `01-VISION.md`, `02-DESIGN-PRINCIPLES.md`.
- **The domain model** — `03-DOMAIN-MODEL.md`.
- **Coding rules and non-negotiables** — `CLAUDE.md`, at the repository root. Read it; the choices below all trace back to it.
- **What is actually built vs. still planned, phase by phase** — `TASKS.md`, at the repository root. That file is the living status tracker; this document does not restate it, because a status list duplicated in two places is how documentation goes stale.

Where an earlier draft of this document proposed a technology, folder, or process the project never adopted, this revision replaces it with what is actually there — not a superset of both.

---

## 1. Backend Stack

| Concern | Choice |
|---|---|
| Language | Go |
| HTTP router | [chi](https://github.com/go-chi/chi) (`github.com/go-chi/chi/v5`) |
| Database driver | [pgx](https://github.com/jackc/pgx) (`github.com/jackc/pgx/v5`) |
| Database | PostgreSQL |
| Migrations | [goose](https://github.com/pressly/goose) (`github.com/pressly/goose/v3`) |
| Auth tokens | [golang-jwt](https://github.com/golang-jwt/jwt) (`github.com/golang-jwt/jwt/v5`) |
| Password hashing | `golang.org/x/crypto` |
| Logging | stdlib `log/slog` (see `internal/log`) |

Notably absent, and deliberately so: no message queue, no cache layer, no metrics/tracing library. Workflow execution is synchronous (see `05-WORKFLOW-ENGINE.md`'s implementation-status note) and nothing in the current system yet needs a queue or a cache — CLAUDE.md's "avoid unnecessary abstractions" applies to infrastructure, not just code. OpenTelemetry and structured metrics are real future needs (see `TASKS.md`'s Production phase) but are not adopted ahead of that need.

## 2. Frontend Stack

| Concern | Choice |
|---|---|
| Framework | Vue 3 (Composition API, per CLAUDE.md's Frontend Guidelines) |
| Language | TypeScript |
| Build tool | Vite |
| Routing | vue-router |
| Icons | `@lucide/vue` |
| Testing | Vitest + `@vue/test-utils` + jsdom |

Also deliberately absent: no state-management library (Zustand/Pinia/Redux), no data-fetching library (TanStack Query/SWR), no UI component kit (shadcn/ui or similar), no CSS framework (Tailwind), no form library (Vue Hook Form/Zod), no rich text/code editor. Global state that genuinely needs to be app-owned (auth, theme, sidebar) is a small hand-written composable holding module-scope refs — see `useAuth.ts` for the pattern. Every form is a plain Vue `ref` per field, validated by the same `internal/platform/validate`-shaped error reporting the backend already returns, surfaced inline. This is not an oversight; it is CLAUDE.md's "avoid unnecessary abstractions" and "small packages" applied to dependency choices — none of the above libraries currently solve a problem this frontend actually has. Add one when a real problem justifies it, not ahead of it.

## 3. Repository Structure

Go code lives at the repository root, not under a `backend/` subdirectory — there is no monorepo split, this is a single Go module with an adjacent `frontend/`.

```
palladium-oss/
  cmd/
    server/       # the long-running API process
    migrate/      # runs goose migrations (up/status)
    bootstrap/    # one-time: creates the first administrator account
    seed/         # one-time, idempotent: populates demo data
  internal/
    <domain>/               # e.g. customer, service, inventory, workflow, event, plugin, authz...
      model.go               # domain types
      validate.go             # domain validation
      repository.go            # repository interface the domain depends on
      httpapi/                # HTTP handlers + request/response DTOs
      postgres/                # the repository interface's Postgres implementation
      service/                 # business logic between httpapi and postgres
    server/        # router.go: mounts every domain's httpapi behind auth + authz middleware
    authz/         # RBAC capability middleware, one Read/Write pair per domain (or domain group)
    auth/          # JWT issuance/validation, user model
    platform/       # small cross-cutting packages: apperror, validate, clock, id, retry, encryption, ssh
  database/
    migrations/    # numbered goose SQL files, one per schema change
  frontend/
    src/
      components/
        app/          # AppShell, UserMenu, and other app-chrome components
        base/          # BaseButton, BaseInput, BaseSelect, BaseModal, BaseCard, ... — the design system's primitives
        data-display/  # DataTable, SimpleTable, FactGrid, RelationshipCard, TimelineEntries, ...
        dialogs/       # one *FormDialog.vue per create/edit flow, plus the shared ConfirmationDialog
        workspace/      # DetailWorkspace, WorkspaceHeader, WorkspaceActions — the Detail Workspace shell
      composables/    # use*.ts — app-owned state (useAuth, useTheme) and collection state (useCustomerCollection, ...)
      services/
        <domain>/      # one folder per domain, each holding a *Repository.ts (the only place apiFetch is called for that domain)
        api/           # httpClient.ts — the single apiFetch seam every repository builds on
      router/         # navigation.ts (NAV_ITEMS) + index.ts (routes, including the auth guard)
      types/          # one file per domain type
      views/          # *CollectionView.vue / *DetailView.vue per workspace
  docker-compose.yml    # local PostgreSQL only
  Makefile
```

Not every domain has every layer — a package only gains a layer once something needs it. `internal/customer` is a complete example of every layer at once; smaller domains may only have `httpapi`/`postgres` if their service layer would just forward calls with no logic worth a separate file.

No `backend/`, `plugins/`, `sdk/`, `deploy/`, `scripts/`, `examples/`, or `tools/` top-level directories exist. Plugins live inside `internal/plugin/` (see `06-PLUGIN-ARCHITECTURE.md`'s implementation-status note) — there is no separate SDK repository or package today.

## 4. Backend Architectural Pattern

Every domain follows the same three-layer chain, wired together in `cmd/server/main.go` and mounted in `internal/server/router.go`. Walking through Customer end to end:

1. **`internal/customer/model.go`** defines `Customer` and `CustomerStatus`; `validate.go` defines `Customer.Validate()`.
2. **`internal/customer/repository.go`** defines the `CustomerRepository` interface (`Get`, `List`, `Create`, `Update`, `Delete`) that the service layer depends on — never a concrete database type.
3. **`internal/customer/postgres`** implements that interface against `pgx`, translating Postgres errors into the shared `apperror` kinds (`not_found`, `conflict`, `invalid`, ...).
4. **`internal/customer/service`** holds the one piece of real business logic this domain has: validating before persisting. It depends on the `CustomerRepository` interface, never the concrete Postgres type, so it is testable with a fake.
5. **`internal/customer/httpapi`** is the REST surface: request/response DTOs distinct from the domain type (so the wire format never leaks internal field layout), a handler per verb, depending on a small local interface satisfied structurally by `*service.CustomerService`.
6. **`internal/server/router.go`** mounts `/customers` behind `auth.Middleware` (valid JWT required) and then `authz.Middleware`'s per-verb capability check (`RequireCustomerRead()` for GET, `RequireCustomerWrite()` for POST/PUT/DELETE).

Every other domain — Service, Location, Device (via `internal/inventory`), Workflow, Event, AccessNetwork, and so on — repeats this exact chain. Reading one domain end to end is reading all of them.

## 5. Frontend Architectural Pattern

The frontend mirrors the backend's layering with its own three pieces, walking through Customer again:

1. **`frontend/src/types/customer.ts`** defines the `Customer` domain type (camelCase), independent of the wire format.
2. **`frontend/src/services/customers/customerRepository.ts`** is the only file that calls `apiFetch` for this domain. It defines a `CustomerDto` (the real snake_case wire shape) and a `fromDto` mapper, and exports `listCustomers`/`getCustomerById`/`createCustomer`/`deleteCustomer`. Where a `List` endpoint has no server-side filtering (most of them, today), search/filter/sort/pagination happen client-side over the full fetched list here — not in the composable or the view.
3. **`frontend/src/composables/useCustomerCollection.ts`** owns the Collection Workspace's state: the current filters, sort, and page as refs, a `watch` that refetches on filter change (resetting to page 1) or page change, and `loading`/`error`/`retry`. It calls the repository, never `apiFetch` directly.
4. **`frontend/src/views/CustomerCollectionView.vue`** and **`CustomerDetailView.vue`** compose the above with the shared `components/data-display` and `components/workspace` primitives. Create/delete flows go through a `components/dialogs/*FormDialog.vue` and the shared `ConfirmationDialog.vue`.

A handful of domains (Location, Service, when managed as a nested resource inside a parent Detail view) skip step 3 — there is no Collection View for them, so there is no composable, just a direct repository call inside the parent view's own `load()` function. Add a composable only once a domain actually gets its own Collection View.

## 6. Authentication & Authorization

- **Authentication**: JWT access tokens only, issued by `POST /auth/login` (see `internal/auth`). There is no refresh token, no logout endpoint, and no password-reset flow yet — those are explicitly out of scope for the current milestone, not an oversight (see `internal/server/router.go`'s own comment on the `/auth` route group). On the frontend, `useAuth.ts` holds the token as module-scope singleton state, decodes its claims client-side for display, and `services/api/httpClient.ts` attaches it as a Bearer header to every request except `/auth/login` itself, clearing it on a 401.
- **Authorization**: Role-Based Access Control via `internal/authz`, one capability pair per domain (or closely related group of domains — e.g. AccessNetwork+OLT+PONPort share one pair since "who can read/write" is the same question at three nested levels of one domain). Enforced per-route in `internal/server/router.go` as middleware, always after `auth.Middleware`, never instead of it.
- **CORS**: a single configurable `AllowedOrigin` (`internal/config`), since the frontend and API run on different ports in local development.

OIDC/OAuth2, external identity providers (LDAP/Active Directory/Entra), and ABAC are real future directions (CLAUDE.md's Architecture section implies growth room here) but are not implemented — do not assume they exist when reading code or writing docs.

## 7. Database Conventions

- UUID primary keys, explicit foreign keys, every schema change through a goose migration under `database/migrations/` — all per CLAUDE.md's Database Rules.
- **Deletes are hard `DELETE` statements**, guarded by `ON DELETE RESTRICT` foreign keys throughout — deleting a Customer that still has a Location fails with a `conflict` error, not a cascade and not a soft-delete. No table has a `deleted_at` column.
- Historical record-keeping — CLAUDE.md's "soft-delete when historical records matter" — is handled instead by `internal/event`: an append-only, immutable Event log keyed by `entity_type`/`entity_id`, written whenever something significant happens to a record (most concretely today, every Workflow transition). A row's own deletion does not need to be undoable for its history to survive; the Event log is that history. If a future domain has a real need to un-delete a specific row (not just know that it existed), that domain can add its own soft-delete column — nothing here forbids it, it just hasn't been needed yet.

## 8. Testing Strategy

Testing is not optional (CLAUDE.md: "Write tests").

- **Backend**: standard library `testing`, no third-party test framework. Tests live next to the code they test (`*_test.go` in the same package) — nearly every `service`, `httpapi`, and `postgres` package has one. `make test` runs the full suite; `make test-integration` additionally runs tests behind the `integration` build tag (real database round-trips, as opposed to fakes).
- **Frontend**: Vitest + `@vue/test-utils`, configured directly in `frontend/vite.config.ts` (no separate `vitest.config.ts`). `npm run test` (from `frontend/`) runs the suite. Tests live next to the file they test (`*.test.ts`). Three layers, each with an established reference pattern worth copying rather than reinventing:
  - **Repository**: `frontend/src/services/customers/customerRepository.test.ts` — mock `apiFetch`, verify client-side filter/sort/pagination and the exact wire shape sent to the API.
  - **Collection composable**: `frontend/src/composables/useCustomerCollection.test.ts` — mock the repository, verify state orchestration (page resets on filter change, sort toggling, loading/error).
  - **Dialog component**: `frontend/src/components/dialogs/DeviceFormDialog.test.ts` — mount for real, mock only the repository; also the reference for testing anything that renders through `BaseModal`'s `<Teleport to="body">` (query via `DOMWrapper(document.body)`, `enableAutoUnmount`).

There is no end-to-end test suite (browser-driven, full stack) today.

## 9. Local Development

```
docker compose up -d postgres   # or: make db-up
make migrate-up
make bootstrap                  # creates the first administrator account, interactively
make seed                       # optional: populates a small demo dataset
make run                        # go run ./cmd/server
```

```
cd frontend
npm install
npm run dev
```

Other Makefile targets: `make build` (builds all four `cmd/` binaries into `bin/`), `make test` / `make test-integration`, `make vet`, `make fmt`, `make tidy`, `make migrate-status`, `make db-down`, `make clean`.

`docker-compose.yml` runs PostgreSQL only (`postgres:16-alpine`) — the API and frontend both run directly on the host, not in containers, during development. There is no CI pipeline today (no `.github/workflows/`); `make vet && make test && make build` (backend) and `npm run build && npm run test` (frontend) are the checks a change is expected to pass before it's considered done, run by hand or by whoever/whatever is making the change.

## 10. Where to Go Next

- **Philosophy and non-negotiables**: `CLAUDE.md` (repository root).
- **Current build status, phase by phase**: `TASKS.md` (repository root).
- **What each entity means and how they relate**: `03-DOMAIN-MODEL.md`.
- **Workflow engine design (vision) vs. what's actually implemented**: `05-WORKFLOW-ENGINE.md`.
- **Plugin architecture design (vision) vs. what's actually implemented**: `06-PLUGIN-ARCHITECTURE.md`.
- **UI architecture and design system**: `07-UI-ARCHITECTURE.md`, `08-DESIGN-SYSTEM.md`, `11-COMPONENT-ARCHITECTURE.md`.
- **Per-workspace field/section specifications**: `09-WORKSPACE-SPECIFICATIONS.md`.

---

# Revision History

| Version | Date | Description |
|---------|------|-------------|
| 1.0 Draft | 2026-07-29 | Initial implementation roadmap |
| 2.0 | 2026-09-03 | Full rewrite to describe the actual stack, structure, and patterns in use, after the original draft's proposed stack/structure/roadmap diverged from what was actually built |

---

**End of Document**
