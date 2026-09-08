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
- [x] ONUs
- [ ] VLANs (explicitly out of scope -- see below)
- [ ] IP pools (explicitly out of scope -- see below)

Covers AccessNetwork -> OLT -> PONPort -> AccessInterface -> AccessAttachment, full CRUD, both backend and frontend (Network Collection View + a Detail Workspace per level, Attach/Detach for equipment).

An ONU is a customer-premises inventory.Device (internal/inventory), not a new domain -- OLTs/switches stay under Network, CPE stays under Devices. "Discover ONU" (Device Collection View) is the built picker: internal/diagnostics/kontron's blacklist scan finds a physically-detected, not-yet-authorized serial number and its PON port interface, then internal/provisioning/kontron/service.AuthorizationService authorizes it on the OLT and fills in the Device record in one flow. The mirror action, "Deauthorize ONU" (Device Detail Workspace), removes the ONU's base authorization and retires the Device -- see docs/06-PLUGIN-ARCHITECTURE.md's Implementation Status note.

VLAN/IP addressing is explicitly out of scope for this OSS (2026-09-08, at the user's explicit request): "All VLAN and IP addressing will be handled by outside equipment - routers and such, not part of the scope of this OSS." The two unchecked items above will stay unchecked.

---

## Phase 5 — Customers

- [x] Customers
- [x] Locations
- [x] Contacts
- [x] Service Equipment (Service <-> Device assignment)
- [x] Remove Customer (internal/customer/removal -- cascades Location/Service/equipment to Archived/Inactive/Disconnected/RemovedAt, real OLT teardown when needed, un-ties Devices rather than deleting them)

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
- [x] Kontron plugin (internal/provisioning/kontron/plugin -- ProvisionService/ResumeService/SuspendService/DisconnectService, real SSH commands against a live C16)
- [ ] MikroTik plugin
- [ ] GenieACS plugin

Real, working Kontron/Iskratel C16 device automation now spans both the Plugin interface and several standalone REST actions that deliberately bypass it, by design (see docs/06-PLUGIN-ARCHITECTURE.md's Implementation Status note for the full picture):

- internal/provisioning/kontron/plugin.Plugin — registered in the Registry, handles Provision/Resume/Suspend/Disconnect for real over SSH (applying/removing a Product's Kontron service-profile).
- internal/diagnostics/kontron — read-only diagnostics plus a network-wide ONU blacklist scan.
- internal/provisioning/kontron/service.AuthorizationService ("Discover ONU") and DeauthorizationService ("Deauthorize ONU", which also retires the Device) — Device-scoped one-shot admin actions, not Service lifecycle transitions, so they stay standalone REST rather than going through the Plugin.
- internal/customer/removal.RemovalService ("Remove Customer") calls ServiceProfileService.Remove directly to tear down real OLT state as part of its cascade.

Palladium is not deferring the Capability Model further out of necessity — VLAN/IP addressing is explicitly out of scope for this OSS (handled by outside equipment), so there is no further Kontron config-change surface currently planned beyond what is built.

---

## Phase 9 — Frontend

- [x] Vue application
- [x] Authentication
- [x] Dashboard (real stats/widgets: Customers/Active Services/Devices/Pending Tasks counts, Network Overview, a bounded system-wide recent-Events feed, Pending Tasks list -- Active Alerts/System Health deliberately omitted, no real data to back them)
- [x] Inventory UI (Site/Building/Room/Rack/Device Detail Workspaces, Inventory Collection View, Rack picker on Device -- lives under Administration in the sidebar, not its own top-level nav item, see docs/04-NAVIGATION.md section 4)
- [x] Customer UI
- [x] Workflow UI (Provision/Suspend/Resume actions + workflow history on the Service Detail Workspace; no dedicated workflow-instance browser)
- [x] Network UI (AccessNetwork -> OLT -> PONPort -> AccessInterface, plus Attach/Detach for equipment)
- [x] Frontend test suite (Vitest + @vue/test-utils)
- [x] Collection View default filters (Customers/Devices/Users hide Archived/Retired-Disposed/Inactive by default, with an "Include..." checkbox -- see docs/09-WORKSPACE-SPECIFICATIONS.md section 5)

There is no standalone Services page or nav entry (2026-09-08, at the user's explicit request) -- a Service is only ever reached through its Customer or Device, never browsed on its own; ServiceDetailView itself is unaffected.

---

## Phase 10 — Production

- [x] CI pipeline (GitHub Actions: backend build/vet/test/integration-test, frontend build/test, on every push/PR to main)
- [ ] Metrics
- [ ] OpenTelemetry
- [ ] Backups
- [ ] HA deployment
- [ ] Kubernetes manifests
