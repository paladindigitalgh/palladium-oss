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
- [x] Buildings
- [x] Rooms
- [x] Racks
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

Current intent for "ONUs" above: an ONU is a customer-premises inventory.Device (internal/inventory), not a new domain -- OLTs/switches stay under Network, CPE stays under Devices. See internal/diagnostics/kontron's blacklist scan and internal/provisioning/kontron's ONU authorization, which already produce exactly what a "New Device" flow needs (a confirmed-online serial number and its assigned PON port interface) to fill in a Device record, once that picker UI is built.

---

## Phase 5 — Customers

- [x] Customers
- [x] Locations
- [x] Contacts
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
- [x] Job queue (internal/workflow/worker.Worker polls for Pending instances; no more synchronous /execute endpoint)
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

Only a simulated mock plugin (internal/plugin/mock) exists today; no real vendor plugin has been built through the Plugin interface yet. Real, working Kontron/Iskratel C16 device automation does exist outside that interface, though: read-only diagnostics plus a network-wide ONU blacklist scan (internal/diagnostics/kontron) and a first real config-change command, ONU authorization (internal/provisioning/kontron) -- see docs/06-PLUGIN-ARCHITECTURE.md's Implementation Status note. Still ahead: VLAN/service-profile assignment (varies per product/plan, a separate step from authorization) and wiring any of this through the actual Plugin/Capability system above.

---

## Phase 9 — Frontend

- [x] Vue application
- [x] Authentication
- [x] Dashboard (real stats/widgets: Customers/Active Services/Devices/Pending Tasks counts, Network Overview, a bounded system-wide recent-Events feed, Pending Tasks list -- Active Alerts/System Health deliberately omitted, no real data to back them)
- [x] Inventory UI (Site/Building/Room/Rack/Device Detail Workspaces, Inventory Collection View, Rack picker on Device)
- [x] Customer UI
- [x] Workflow UI (Provision/Suspend/Resume actions + workflow history on the Service Detail Workspace; no dedicated workflow-instance browser)
- [x] Network UI (AccessNetwork -> OLT -> PONPort -> AccessInterface, plus Attach/Detach for equipment)
- [x] Frontend test suite (Vitest + @vue/test-utils)

---

## Phase 10 — Production

- [x] CI pipeline (GitHub Actions: backend build/vet/test/integration-test, frontend build/test, on every push/PR to main)
- [ ] Metrics
- [ ] OpenTelemetry
- [ ] Backups
- [ ] HA deployment
- [ ] Kubernetes manifests
