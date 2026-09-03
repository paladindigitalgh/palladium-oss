# Palladium OSS Development Roadmap

## Phase 0 — Foundation

- [x] Initialize Go application
- [x] Create HTTP server
- [x] Graceful shutdown
- [x] Structured logging
- [x] Configuration loading
- [x] Health endpoint
- [x] Readiness endpoint
- [x] Dependency injection
- [x] Basic middleware
- [x] Docker development environment

---

## Phase 1 — Database

- [x] PostgreSQL connection
- [x] Goose migrations
- [x] Base schema
- [x] Migration tooling
- [x] Repository layer

---

## Phase 2 — Authentication

- [x] User model
- [x] Roles
- [x] Permissions (RBAC capabilities, see internal/authz)
- [x] JWT
- [ ] OIDC
- [x] Login API

---

## Phase 3 — Inventory

Hierarchy is Site -> Building -> Room -> Rack -> Device (internal/inventory).

- [x] Sites
- [ ] Buildings
- [ ] Rooms
- [ ] Racks
- [x] Devices
- [ ] Inventory history

---

## Phase 4 — Network

- [x] OLTs
- [x] PON ports
- [ ] Splitters
- [ ] ONUs
- [ ] VLANs
- [ ] IP pools

Covers AccessNetwork -> OLT -> PONPort -> AccessInterface -> AccessAttachment, full CRUD, both backend and frontend (Network Collection View + a Detail Workspace per level, Attach/Detach for equipment).

---

## Phase 5 — Customers

- [x] Customers
- [x] Locations
- [ ] Contacts
- [x] Service Equipment (Service <-> Device assignment)

---

## Phase 6 — Services

- [x] Products
- [ ] Packages
- [x] Service lifecycle

---

## Phase 7 — Workflow Engine

- [x] Workflow model
- [x] Task execution (engine dispatches to plugin capabilities)
- [ ] Job queue (execution is synchronous today, no async queue)
- [x] Retry logic
- [x] Audit trail (Event domain)

---

## Phase 8 — Plugins

- [x] Plugin SDK (internal/plugin)
- [x] Plugin loader (registry)
- [x] Capability discovery
- [ ] Kontron plugin
- [ ] MikroTik plugin
- [ ] GenieACS plugin

Only a simulated mock plugin (internal/plugin/mock) exists today; no real vendor plugin has been built yet.

---

## Phase 9 — Frontend

- [x] Vue application
- [x] Authentication
- [ ] Dashboard (still placeholder data throughout, not wired to any real query -- see frontend/src/views/DashboardView.vue)
- [x] Inventory UI (Device workspace; Building/Room/Rack have no UI, matching Phase 3)
- [x] Customer UI
- [x] Workflow UI (Provision/Suspend/Resume actions + workflow history on the Service Detail Workspace; no dedicated workflow-instance browser)
- [x] Network UI (AccessNetwork -> OLT -> PONPort -> AccessInterface, plus Attach/Detach for equipment)
- [x] Frontend test suite (Vitest + @vue/test-utils)

---

## Phase 10 — Production

- [ ] Metrics
- [ ] OpenTelemetry
- [ ] Backups
- [ ] HA deployment
- [ ] Kubernetes manifests
