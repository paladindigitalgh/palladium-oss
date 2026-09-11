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

- [x] User model (gained optional First Name/Last Name, 2026-09-10 -- see docs/03-DOMAIN-MODEL.md section 25)
- [x] Roles
- [x] Permissions (RBAC capabilities, see internal/authz)
- [x] JWT
- [ ] OIDC
- [x] Login API
- [x] Self-service Profile (2026-09-10): a User edits their own First/Last Name and password via a Profile screen reached from the account menu, guarded by `/me` (any authenticated Role) rather than the Administrator-only `/users` API User Management uses

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
- [ ] Connection Profile management UI (create/edit)
- [ ] VLANs (explicitly out of scope -- see below)
- [ ] IP pools (explicitly out of scope -- see below)

Covers AccessNetwork -> OLT -> PONPort -> AccessInterface -> AccessAttachment, full CRUD, both backend and frontend (Network Collection View + a Detail Workspace per level, Attach/Detach for equipment).

An ONU is a customer-premises inventory.Device (internal/inventory), not a new domain -- OLTs/switches stay under Network, CPE stays under Devices. Authorizing one from the OLT's blacklist scan is no longer a separate "Discover ONU" step (2026-09-09) -- New Device's own Serial Number field can be pointed at internal/diagnostics/kontron's blacklist scan (a physically-detected, not-yet-authorized serial number and its PON port interface) instead of typed by hand, and internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService authorizes it on the OLT and creates the Device in that one action. Since 2026-09-10, that same action also finds-or-creates the matching PONPort/AccessInterface, so the Access Network topology needed for later provisioning already exists with no manual Network-workspace step. The mirror action, "Remove Device" (Device Workspace, renamed 2026-09-09 from "Deauthorize ONU" -- same behavior), removes the ONU's base authorization and retires the Device -- see docs/06-PLUGIN-ARCHITECTURE.md's Implementation Status note.

VLAN/IP addressing is explicitly out of scope for this OSS (2026-09-08, at the user's explicit request): "All VLAN and IP addressing will be handled by outside equipment - routers and such, not part of the scope of this OSS." The two unchecked items above will stay unchecked.

Connection Profile (internal/connectionprofile) has a complete backend domain and REST API but no frontend UI to create or edit one -- only OLTFormDialog.vue's picker (added 2026-09-08) to *assign* an existing profile to an OLT. A fresh environment with zero Connection Profiles has no in-product way to make its first one; someone has to POST /connection-profiles directly. Deferred rather than built alongside the picker since an existing profile already covered the immediate need.

---

## Phase 5 — Customers

- [x] Customers
- [x] Locations
- [x] Contacts
- [x] Service Equipment (Service <-> Device assignment)
- [x] Customer Device (internal/customerdevice, added 2026-09-09 -- Customer <-> Device placement, independent of any Service; the one deliberate exception to "equipment is associated through Services," see docs/03-DOMAIN-MODEL.md section 26. Attach/Detach live on the Customer Workspace's own Devices section. Gained an optional Location field 2026-09-10 -- which of the Customer's own Locations the Device sits at, for tracking purposes only; a side effect is that a Location can never be hard-deleted once any Device was ever placed there, even long after detached, since that history row is never erased -- internal/location's delete-conflict error now names this specifically instead of guessing "still has a Service attached")
- [x] Remove Customer (internal/customer/removal -- cascades Location/Service/equipment to Archived/Inactive/Disconnected/RemovedAt, real OLT teardown when needed, un-ties Devices rather than deleting them; does not yet detach active Customer Device placements -- a known gap, see docs/03-DOMAIN-MODEL.md section 26)
- [x] Notes (internal/note, added 2026-09-10 -- free-text, operator-authored, immutable commentary attached to a Customer, Device, or Service; newest-first, paginated Notes section on all three Detail Workspaces, author/timestamp always shown)

CustomerType dropped its "Government" value 2026-09-10 (Residential, Business, and Internal are the three that remain) -- modeled since early on but never actually used.

---

## Phase 6 — Services

- [x] Products
- [ ] Packages
- [x] Service lifecycle

Adding a Service from the Customer Workspace (2026-09-09) is now device-specific and self-provisioning, not three separate steps: it is disabled until the Customer has an eligible Device (attached, not already fulfilling a different Service), defaults to Active status with no Status field shown, ties the chosen Device via a real Service Equipment record, and runs the real provision-service Workflow against it in the same action -- see docs/09-WORKSPACE-SPECIFICATIONS.md section 8. Creating that Service Equipment record now also auto-syncs an Access Attachment (2026-09-10, `ServiceEquipmentService.syncAccessAttachment`) whenever the Device has a known `OnuAuthorization` and matching Access Interface on file -- closing the gap where an operator had to build one by hand in the Network workspace first, for any Device authorized through Palladium's own OLT blacklist flow (see docs/03-DOMAIN-MODEL.md section 7). Since 2026-09-10, Add Service is also all-or-nothing: a failed provisioning attempt deletes the Service and its Service Equipment record again rather than leaving an unprovisioned one behind, with the error shown inline in the still-open dialog instead of a dismissible banner. The Device picker itself is shown whenever the Customer has more than one attached Device at all, not just more than one *eligible* one.

Removing a Service (2026-09-10) has two entry points: the Service Workspace's own Remove Service is strict (attempts the delete, shows the backend's conflict message if blocked); the Customer Workspace's "Remove" link on a Service row does the whole teardown in one action -- suspends it on the OLT first if Active, detaches every assigned equipment record, then deletes the Service.

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
- internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService (composes AuthorizationService's AuthorizeONU with Device creation, triggered inline from New Device -- no longer a standalone "Discover ONU" step, see docs/06-PLUGIN-ARCHITECTURE.md) and DeauthorizationService ("Remove Device" in the UI, renamed 2026-09-09 from "Deauthorize ONU"; also retires the Device) — Device-scoped one-shot admin actions, not Service lifecycle transitions, so they stay standalone REST rather than going through the Plugin.
- internal/customer/removal.RemovalService ("Remove Customer") calls ServiceProfileService.Remove directly to tear down real OLT state as part of its cascade.

Palladium is not deferring the Capability Model further out of necessity — VLAN/IP addressing is explicitly out of scope for this OSS (handled by outside equipment), so there is no further Kontron config-change surface currently planned beyond what is built.

---

## Phase 9 — Frontend

- [x] Vue application
- [x] Authentication
- [x] Dashboard (real stats/widgets: Customers/Active Services/Devices/Pending Tasks counts, Network Overview, a bounded system-wide recent-Events feed, Pending Tasks list -- Active Alerts/System Health deliberately omitted, no real data to back them)
- [x] Inventory UI (Site/Building/Room/Rack/Device Detail Workspaces, Inventory Collection View, Rack picker on Device -- lives under Administration in the sidebar, not its own top-level nav item, see docs/04-NAVIGATION.md section 4). Device Collection is CPE-scoped (2026-09-08): New Device no longer asks for Rack/Asset Tag/Status (defaults every new Device to Unused), and status itself collapsed to a flat Unused/Active/Retired set, Active now derived from either a Customer Device placement or a Service Equipment assignment. "Delete Device" was removed entirely (2026-09-09, no delete action exists) in favor of "Remove Device" (renamed from "Deauthorize ONU," same behavior); Remove Device now navigates back to the Devices list on success (2026-09-10). Device Manufacturer/Model on the Administration Hardware panel gained a per-entry Default checkbox (2026-09-10) that New Device pre-selects from
- [x] Customer UI, now including a Devices section (2026-09-09 -- Attach/Detach a Device directly to a Customer, internal/customerdevice) and Add Service gated on having an eligible Device. The Services table gained a Device column and ONU Diagnostics blocks became individually collapsible (2026-09-10)
- [x] Workflow UI (Provision/Suspend/Resume actions + workflow history on the Service Detail Workspace; no dedicated workflow-instance browser). Provision-service also runs automatically now from the Customer Workspace's Add Service (2026-09-09), not only this manual button
- [x] Network UI (AccessNetwork -> OLT -> PONPort -> AccessInterface, plus Attach/Detach for equipment)
- [x] Notes UI (internal/note -- collapsible, paginated, newest-first Notes section with an inline add-note textarea, on the Customer, Device, and Service Detail Workspaces, added 2026-09-10). Notes (and the account menu) now show an author's First/Last Name when set, falling back to Email otherwise (2026-09-10)
- [x] Explorer UI (2026-09-11 -- docs/09-WORKSPACE-SPECIFICATIONS.md section 15, docs/03-DOMAIN-MODEL.md section 28): three curated cross-domain reports (Customers & Contacts, Devices, Customers & Devices) backed by real internal/report SQL joins, not a dynamic ad hoc query builder. A report is fetched once and searched/sorted/exported to CSV entirely client-side; a result row opens into its own Customer or Device Detail Workspace
- [x] Frontend test suite (Vitest + @vue/test-utils)
- [x] Collection View default filters (Customers/Devices/Users hide Archived/Retired-Disposed/Inactive by default, with an "Include..." checkbox -- see docs/09-WORKSPACE-SPECIFICATIONS.md section 5)

SectionCard's collapse animation (every Detail Workspace section, app-wide) was fixed 2026-09-10: it was failing to reach a true 0 height whenever padding lived directly on the collapsing element itself, since padding can never shrink below its own size -- padding now lives on an inner wrapper instead, the same fix BaseDisclosure.vue already had.

A disabled BaseButton's accessible disabled-reason text (a `position: absolute` visually-hidden span) had no positioned ancestor anywhere up to `<html>`, so its fallback position was computed against the whole document's unclipped layout instead of the button -- silently making the entire app scrollable past its real content on any page with a disabled button that had a reason (found live on the Customer Workspace's disabled "Add Service" button). Fixed 2026-09-10 by giving `.base-button` `position: relative`.

There is no standalone Services page or nav entry (2026-09-08, at the user's explicit request) -- a Service is only ever reached through its Customer or Device, never browsed on its own; ServiceDetailView itself is unaffected.

---

## Phase 10 — Production

- [x] CI pipeline (GitHub Actions: backend build/vet/test/integration-test, frontend build/test, on every push/PR to main)
- [ ] Metrics
- [ ] OpenTelemetry
- [ ] Backups
- [ ] HA deployment
- [ ] Kubernetes manifests
