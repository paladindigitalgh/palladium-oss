---
document: 03-DOMAIN-MODEL
status: Draft
title: Domain Model
version: 1.9-draft
---

# Domain Model

## Executive Summary

The Domain Model defines the core business concepts that make up
Palladium. These concepts are independent of databases, APIs, vendors,
and user interfaces. They represent how the business understands its
operational world.

A well-defined domain model ensures that every subsystem speaks the same
language. User interfaces, workflows, database schemas, plugins, and
APIs should all be built upon these shared concepts.

This document describes the primary entities, their responsibilities,
ownership, and relationships.

------------------------------------------------------------------------

# Table of Contents

1.  Purpose
2.  Design Philosophy
3.  Core Domain Concepts
4.  Customer
5.  Service
6.  Device
7.  Service Equipment
8.  Site
9.  Relationship Overview
10. Workflow
11. Workflow Instance
12. Event
13. Vendor Plugin
14. Transport
15. Domain Boundaries
16. Entity Lifecycles
17. Domain Invariants
18. Future Domain Expansion
19. Contact

------------------------------------------------------------------------

# 1. Purpose

The purpose of the domain model is to define *what Palladium knows*.

It does **not** define:

-   Database tables
-   API endpoints
-   User interfaces
-   Vendor implementations

Instead, it establishes a common vocabulary for the entire platform.

Every feature should reference these domain concepts rather than
inventing new terminology.

------------------------------------------------------------------------

# 2. Design Philosophy

The domain model follows several fundamental rules.

## Business Before Technology

Entities represent business concepts rather than technical
implementation details.

For example, a **Service** exists regardless of whether it is
provisioned on a Kontron OLT, a Nokia OLT, or an Active Ethernet switch.

Technology changes.

Business concepts remain stable.

## Stable Core, Flexible Edge

Core entities should change very slowly.

Vendor-specific behavior belongs inside plugins and adapters rather than
the domain model.

## Explicit Relationships

Relationships should be modeled intentionally.

Avoid hidden or implied relationships.

Ownership should always be obvious.

------------------------------------------------------------------------

# 3. Core Domain Concepts

The core domain of Palladium consists of the following primary entities.

-   Customer
-   Service
-   Device
-   Service Equipment
-   Site
-   Workflow
-   Event
-   Vendor Plugin

Supporting entities will be introduced in later sections as required.

------------------------------------------------------------------------

# 4. Customer

A Customer represents the individual or organization receiving one or
more network services.

The Customer is primarily a business entity.

Responsibilities include:

-   Identity
-   Contact information (see section 19, Contact -- resolved on demand,
    not a field on Customer itself)
-   Status
-   Billing reference (external)
-   Associated services

A Customer **owns Services**.

Customers do **not** directly own network equipment. Equipment is
associated through services — with one deliberate, narrow exception: see
section 26 (Customer Device), added 2026-09-09 to model a Device
physically placed at a Customer's premises before, or entirely without,
an active Service (a real install can precede activation by days). That
exception never grows into a general "Customer owns Devices" model — it
is a placement record, not an ownership one, and a Device attached this
way still carries no Service, billing, or provisioning meaning until a
real Service Equipment link exists for it.

------------------------------------------------------------------------

# 5. Service

A Service represents a deliverable network offering.

Examples include:

-   Residential Internet
-   Business Internet
-   Dedicated Ethernet
-   Transport circuit

A Service contains operational characteristics such as:

-   Service plan
-   Operational status
-   Provisioning state
-   Assigned equipment
-   Service address

Services are the operational bridge between customers and the physical
network.

------------------------------------------------------------------------

# Architect's Note

A customer buys services.

Services use equipment.

Keeping these concepts separate greatly simplifies long-term system
evolution.

# 6. Device

A Device represents a managed piece of physical equipment that
Palladium is capable of identifying, tracking, and operating.

Device is deliberately generic and vendor-agnostic. It carries no
type/subtype distinction — Manufacturer, Model, and Serial Number are
plain fields on the one Device concept, not a taxonomy of device kinds.
Function- and vendor-specific concepts (OLTs, routers, switches, cards,
ports, splitters, ...) are out of scope for Device itself; they are
either their own domain (see OLT below) or belong to a future,
more-specific model layered on top of Device once one is needed.

Note: OLT is **not** a kind of Device. It is a separate domain, itself
the root of the Network hierarchy (OLT → PON Port → Access Interface →
Access Attachment) with no relationship to Device or Site at all — see
the Site section below.

## Responsibilities

A Device is responsible for maintaining:

-   Identity
-   Manufacturer
-   Model
-   Serial number
-   Asset tag
-   Operational status
-   Physical location (optional — a Device may exist unracked)
-   Current assignment

Devices exist independently of customers, with the one narrow exception
of a Customer Device placement (section 26).

A Device sitting in a warehouse, not yet racked or assigned, is still a
Device.

Operational status (2026-09-09, revised) is a flat, three-value set —
Unused, Active, Retired — not the longer procurement-style progression
this document originally described (see section 16's Device row for the
correction). Active means "currently in use," which as of the Customer
Device addition (section 26) can now come from either of two
independent sources: an active Customer Device placement, or an active
Service Equipment assignment (section 7) — a Device stays Active as long
as *either* is true, and only drops to Unused once *both* are gone.
Retired is terminal and reached only by fully deauthorizing a Device
from its OLT (see docs/06-PLUGIN-ARCHITECTURE.md); nothing ever
un-retires a Device automatically. A Device is never permanently
deleted — there is no delete action for it at all, by design (see
section 17's corresponding invariant); "removing" a Device from service
always means transitioning its status, never erasing the row.

------------------------------------------------------------------------

# 7. Service Equipment

Service Equipment represents the relationship between a Service and one
or more Devices.

This entity exists because equipment assignments change over time while
the equipment itself remains the same.

Examples include:

-   ONU replacement
-   Router replacement
-   Temporary loan equipment
-   Hardware upgrades

By modeling assignments separately, Palladium preserves historical
accuracy while allowing equipment to move between services throughout
its lifecycle.

## Responsibilities

Service Equipment records:

-   Assigned service
-   Assigned device
-   Assignment date
-   Removal date
-   Assignment reason
-   Current status

Only one active assignment should exist for a given device at a time.

Removing an assignment is normally a soft operation: a removal date and
reason are recorded rather than deleting the row, so historical
assignments remain queryable. A real hard delete also exists
(`DELETE /service-equipment/{id}`) for disposable test/demo data an
operator is deliberately cycling through — it is blocked by an active
Access Attachment (foreign-key `RESTRICT`) until that attachment is
removed first, and the frontend only offers it as a distinct "Remove"
action, never a replacement for the soft path.

Removing a Device from its OLT ("Remove Device" on the Device Workspace
— see docs/06-PLUGIN-ARCHITECTURE.md) needs to resolve which OLT and
interface a Device is authorized on, which normally comes from its
Service Equipment record's Access Attachment. A Device authorized
directly (picked from the OLT's blacklist scan while creating it, before
any Service exists for it at all) has no Service Equipment record yet to
resolve that from — `internal/onuauthorization`'s `OnuAuthorization`
record is the fallback: a lean, OLT-scoped record of exactly which
interface a Device was authorized on, written the moment authorization
succeeds and consulted only when no Service Equipment record exists to
answer the question instead.

Since 2026-09-10, creating a Service Equipment record also auto-syncs an
Access Attachment for it, closing what was previously a manual gap: an
operator authorizing a Device directly from an OLT's blacklist scan (the
same flow the `OnuAuthorization` paragraph above describes) used to have
to separately build a PON Port, Access Interface, and Access Attachment
by hand in the Network workspace before that Device could ever be
provisioned through a Service — real provisioning
(`internal/accesstopology.Resolver.Locate`) has nowhere to send config
without one. This is now two cooperating steps, split along the same
vendor/core boundary this document's Design Philosophy (section 2)
requires everywhere else:

-   **Vendor-specific half** (`internal/provisioning/kontron/service`):
    once `AuthorizeAndCreateDeviceService` records the `OnuAuthorization`
    for a freshly authorized Device, it also finds-or-creates the PON
    Port and Access Interface for the interface the OLT actually
    assigned — parsing that Kontron-shaped interface string (e.g.
    `xgs/1/3`) stays confined to this plugin package, never leaking into
    a vendor-agnostic one.
-   **Vendor-agnostic half** (`internal/serviceequipment/service`):
    `ServiceEquipmentService.Create`'s `syncAccessAttachment` step looks
    up the new Service Equipment's Device by `OnuAuthorization`, then by
    Access Interface (a plain ID/string lookup, no vendor-specific
    parsing at all), and if both are found, creates the Access
    Attachment linking them. Either lookup missing is a silent no-op,
    not an error — most Devices have no `OnuAuthorization` at all, and
    this must never block Add Service — which preserves the manual
    Network-workspace fallback for any Device Palladium does not have a
    known network position for (one never authorized through this
    OLT-blacklist flow, for instance).

An operator using Palladium's own "authorize this ONU" flow end-to-end
now never has to touch the Network workspace by hand for that Device.

------------------------------------------------------------------------

# 8. Site

A Site represents a physical location operated by the ISP.

Examples include:

-   Central Office
-   POP
-   Headend
-   Remote Cabinet
-   Data Center

Sites provide operational context for devices, via the Inventory
hierarchy: Site → Building → Room → Rack → Device.

A Site may contain:

-   Buildings, Rooms, and Racks
-   Devices installed in those Racks
-   Supporting equipment

Note: OLT does not belong to this hierarchy at all. It lives in a
separate domain, itself the root of the Network hierarchy, with no
relationship to Site — see the Device section above.

Customers are not located at Sites.

Customer service addresses are modeled separately as part of a Service.

------------------------------------------------------------------------

# 9. Relationship Overview

The core relationships within Palladium are intentionally
straightforward.

Customer → owns one or more Services

Service → has one or more Service Equipment records

Service Equipment → references exactly one Device

Device → optionally belongs to one Site (indirectly, via Rack → Room →
Building → Site — a Device may also exist unracked, belonging to no
Site yet)

Customer → optionally has one or more Devices placed at its premises
directly (Customer Device, section 26), independent of any Service —
the one deliberate exception to this document's general rule that
equipment is only ever associated through Services (see section 4)

This structure separates business relationships from physical
infrastructure, allowing equipment to be reassigned without altering
customer or service history.

Note: OLT has no place in this relationship chain — it roots its own
Network hierarchy, never belongs to a Site, and is never referenced
by Service Equipment (that always references a Device).

------------------------------------------------------------------------

# Design Principle

Model relationships explicitly.

Avoid embedding ownership inside entities when the relationship itself
has operational value, history, or lifecycle.

# 10. Workflow

A Workflow defines a repeatable operational process.

Unlike an Event, which records something that happened, a Workflow
defines *how* an operation should be performed.

Examples include:

-   Provision Service
-   Replace ONU
-   Upgrade Service
-   Suspend Service
-   Run Diagnostics
-   Restore Customer
-   Discover Device

A Workflow is a template.

It contains:

-   Purpose
-   Inputs
-   Validation rules
-   Execution steps
-   Success criteria
-   Failure handling

Workflows should be deterministic whenever possible.

Operators should know what to expect before execution begins.

------------------------------------------------------------------------

# 11. Workflow Instance

A Workflow Instance represents a single execution of a workflow.

For example:

Workflow: Replace ONU

Instances:

-   Customer A replacement on July 8
-   Customer B replacement on July 15
-   Emergency replacement after equipment failure

Each execution records:

-   Start time
-   End time
-   Operator
-   Current status
-   Execution log
-   Results
-   Generated events

Workflow definitions are reusable.

Workflow instances are historical records.

------------------------------------------------------------------------

# 12. Event

An Event records something that has already occurred.

Events are immutable.

Examples include:

-   Customer created
-   Service activated
-   ONU discovered
-   Router synchronized
-   Workflow completed
-   Workflow failed
-   Device assigned

Events provide the chronological history of Palladium.

They should never contain business logic.

Instead, they describe facts.

Future automation, reporting, timelines, and auditing all depend upon a
complete and trustworthy event history.

As of 2026-09-11, most of the examples above are real: Customer created,
Device created, Device retired ("assigned"/"unassigned" in practice means
attached to or detached from a Customer or a Service, not a status field
of its own), Contact added, Location added, Service added/removed, and
Service Equipment attached/detached each write a real Event today (see
`internal/event`'s own package doc comment for the full call-site list).
ONU discovered and Router synchronized remain aspirational — nothing in
this codebase talks to vendor hardware closely enough yet to emit either.
The Explorer Activity page (docs/09-WORKSPACE-SPECIFICATIONS.md section
15) is the operator-facing, searchable view of this same history.

------------------------------------------------------------------------

# Architectural Principle

Workflows create Events.

Events do not execute Workflows.

This one-way relationship keeps execution separate from historical
record and avoids circular dependencies. It describes Workflow's own
relationship to Event specifically, not an exclusivity claim — other
domains (Customer, Device, Contact, Location, Service, Service
Equipment) write their own Events directly too, from their own httpapi
handlers rather than through a Workflow, for actions a Workflow was
never involved in creating in the first place.

------------------------------------------------------------------------

# 13. Vendor Plugin

Vendor Plugins isolate hardware-specific behavior from the core
platform.

A plugin translates Palladium's business intent into vendor-specific
operations.

Examples include:

Business Intent: - Reboot ONU

Plugin: - Execute the appropriate command sequence for Kontron - Execute
the appropriate API request for Nokia - Execute the appropriate SSH
commands for another vendor

The rest of Palladium should never need to know which vendor is
involved.

------------------------------------------------------------------------

# Design Goal

Business logic belongs in Palladium.

Vendor logic belongs in plugins.

# 14. Transport

A Transport is the mechanism used by Palladium to communicate with
external systems and devices.

The Transport layer is intentionally separated from Vendor Plugins.

Examples include:

-   SSH
-   HTTP/HTTPS
-   REST APIs
-   NETCONF
-   SNMP
-   TR-069 / CWMP
-   Serial Console (future)

A Vendor Plugin declares *what* operations must be performed.

The Transport determines *how* those operations are delivered.

This separation allows the same plugin to evolve as communication
methods change without affecting the core domain model.

------------------------------------------------------------------------

# 15. Domain Boundaries

Clear boundaries between domains are essential for long-term
maintainability.

The following responsibilities belong to the core domain:

-   Customers
-   Services
-   Devices
-   Service Equipment
-   Sites
-   Workflows
-   Workflow Instances
-   Events

The following responsibilities are intentionally outside the core
domain:

-   Billing
-   CRM
-   GIS
-   Accounting
-   Monitoring platforms
-   Authentication providers

These systems may integrate with Palladium, but they do not define the
operational model.

------------------------------------------------------------------------

# 16. Entity Lifecycles

Every entity progresses through a lifecycle.

Understanding these lifecycles is critical to preserving data integrity.

## Customer

Active, Inactive, Archived — deliberately a flat, unordered set: a
Customer's status answers "should this identity currently be treated as
active," not "where are they in a provisioning workflow." See
`internal/customer/status.go`.

## Service

Pending → Active → Suspended → Disconnected. This is the typical
progression, not a code-enforced sequence — `internal/service`'s status
type has no transition guard of its own; nothing stops a caller from
setting any valid status directly. The real enforcement lives one layer
up, in `internal/workflow`: the provision-service/suspend-service/
resume-service workflow definitions are what an operator actually
triggers, and each maps to exactly one status change.

## Device

Unused → Active → Retired (revised 2026-09-09; superseded the original
seven-value procurement progression this row described — Ordered →
Received → In Stock → Installed → Maintenance → Retired → Disposed —
which never matched how Device is actually scoped: CPE out in customer
homes and businesses, not shelf/rack inventory, so "on order" /
"received" / "in maintenance" never had real meaning here). Active is
set automatically from two independent sources — an active Customer
Device placement (section 26) or an active Service Equipment assignment
(section 7) — not chosen by an operator; see section 6's own note on
the two-source rule. Retired is one-way and reached only by fully
deauthorizing a Device from its OLT.

## Workflow Instance

Pending → Running → Succeeded \| Failed \| Cancelled. Unlike the three
above, this one **is** code-enforced — see the transition map in
`internal/workflow/status.go`.

## User

Active ⇄ Inactive — a flat, two-value, reversible status, not a
progression: it answers "may this identity currently authenticate,"
nothing more. Users are never deleted, so Inactive is this entity's only
way to record "cannot log in, record kept." See `internal/auth/status.go`.

## Event

Created → Immutable Archive

Entity state changes should always be represented explicitly and should
generate corresponding operational events.

## Note

Created → Immutable Archive — the same shape as Event, for the same
reason (section 27): once an operator writes one, it is never changed or
removed.

------------------------------------------------------------------------

# 17. Domain Invariants

The following rules must always remain true.

-   A Customer may own multiple Services.
-   A Service belongs to exactly one Customer.
-   A Device may exist without being assigned.
-   A Device may have only one active Service Equipment assignment at a
    time.
-   A Device may have only one active Customer Device placement at a
    time (section 26) — the same one-active-record-per-device rule as
    Service Equipment, applied to the Customer-placement side.
-   A Device with an active Service Equipment assignment cannot be
    detached from its Customer Device placement until that assignment
    is removed first (section 26) — detaching never cascades a removal
    onto the Service side.
-   A Device is never permanently deleted, by design — there is no
    delete action for it at all (2026-09-09; superseded the original
    hard-delete action once offered on the Device Workspace). Removing
    a Device from service always means transitioning its status
    (Retired), never erasing the row — the identical "always preserve
    history" reasoning Service Equipment's own soft-removal already
    follows below.
-   Service Equipment assignments are removed (soft) by default, keeping
    historical records queryable; a real delete exists only for
    disposable test/demo data and is blocked while an active Access
    Attachment references the record.
-   Workflow definitions are immutable once published.
-   Workflow Instances are permanent historical records.
-   Events are immutable.
-   Notes are immutable once written — there is no update or delete
    action for one at all (section 27).
-   A Location can never be permanently deleted once a Service has ever
    been sold at it or a Device has ever been placed there, even long
    after both are gone — `services.location_id` and
    `customer_devices.location_id` are both `ON DELETE RESTRICT`, and
    neither a disconnected Service nor a detached Customer Device row is
    ever deleted, only marked so (sections 5 and 26).
-   Vendor Plugins never contain business rules.
-   Users are never deleted; deactivation (status) is the only way to
    revoke a User's ability to log in.

Violating these invariants risks corrupting the operational model.

------------------------------------------------------------------------

# 18. Future Domain Expansion

The domain model has been intentionally designed for growth.

Future entities may include:

-   Maintenance Window
-   Change Request
-   Scheduled Task
-   Alarm
-   Notification
-   Credential Vault Reference
-   Inventory Batch
-   Software Image
-   Firmware Catalog

These additions should extend the existing model rather than replacing
it.

------------------------------------------------------------------------

# 19. Contact

A Contact is a person to reach about a Customer's account -- billing,
technical, or emergency.

This is the domain section 4's "Contact information" responsibility
pointed to: contact information is not a field on Customer itself, the
same separation Location already has for addresses. A Customer's
Contacts are resolved on demand, not embedded.

## Responsibilities

A Contact records:

-   Owning customer
-   Name
-   Role (Primary, Billing, Technical, Emergency, Other)
-   Email
-   Phone
-   Current status

A Customer may have zero or more Contacts. Unlike Location and Service,
a Contact has no further child of its own, and unlike every other
Customer sub-resource, deleting the Customer deletes its Contacts along
with it -- a Contact has no operational significance independent of the
Customer it belongs to.

------------------------------------------------------------------------

# 20. Product Catalog

A Product Catalog groups related Products under one named,
independently-lifecycled collection -- e.g. "Residential Internet"
versus "Business Internet".

## Responsibilities

A Product Catalog records:

-   Name
-   Current status (Active / Inactive)

A Product Catalog describes how the ISP organizes its own offerings,
never a subscriber's actual service, and never a price. It exists
purely to group Products (see section 21) -- it has no other
responsibility.

------------------------------------------------------------------------

# 21. Product

A Product is a single commercial offering the ISP sells -- e.g.
"Residential Internet 500 Mbps" -- described independently of who buys
it and how it is delivered.

## Responsibilities

A Product records:

-   Owning Product Catalog
-   Owning Provider (see section 23)
-   Category (Internet, Voice, IPTV, Transport, Managed WiFi, Other)
-   Name
-   Current status (Active / Retired -- one-way, unlike most other
    status fields in this document)

A Service (section 5) references exactly one Product -- "what was
sold" -- and, separately, exactly one Service Profile (section 22) --
"how it is meant to operate." Neither is derived from the other; both
are required. A Product carries no pricing, no bandwidth profile, and
no vendor-specific configuration of its own -- the OLT profile that
actually delivers it is a separate concern (see section 24,
Provisioning Profile).

------------------------------------------------------------------------

# 22. Service Profile

A Service Profile is a named, reusable description of a Service's
operational intent -- e.g. "Residential Standard" versus "Business
Ethernet" -- distinct from which Product was sold.

## Responsibilities

A Service Profile records:

-   Name
-   Current status (Active / Inactive)

Like Product, a Service Profile carries no bandwidth, QoS, VLAN, or
vendor-specific detail of its own -- that remains a future Network
domain's concern, layered on top of a Service, never folded into its
Service Profile.

------------------------------------------------------------------------

# 23. Provider

A Provider is the retail ISP identity a Product belongs to -- the
company selling it -- distinct from the network operator that owns the
physical OLTs and PON ports it is delivered over.

## Responsibilities

A Provider records:

-   Name
-   Current status (Active / Inactive)

This distinction is invisible in a single-ISP deployment: exactly one
Provider exists, and nothing in the product surfaces it. It becomes
real on an open-access network, where more than one ISP sells service
over one shared physical network -- each Provider's Products, and the
OLT vendor profiles that deliver them (see section 24), stay fully
isolated from one another even when they happen to share an identical
speed tier.

------------------------------------------------------------------------

# 24. Provisioning Profile

A Provisioning Profile maps one Product to the exact configuration
profile a specific OLT vendor already has running for it -- the rate
limiting and VLAN assignment an operator builds by hand directly on the
OLT.

## Responsibilities

A Provisioning Profile records:

-   Owning Product
-   Vendor
-   Profile name

Palladium never generates, applies, or modifies this profile itself --
see section 13 (Vendor Plugin) and section 14 (Transport) for why that
boundary matters; applying a profile to a live ONU is future
provisioning work, layered on top of this lookup, not implied by it.
One Product maps to at most one profile per vendor, and a given
vendor's profile name identifies exactly one Product.

------------------------------------------------------------------------

# 25. User & Role

A User is an authentication identity — someone who can log in to
Palladium itself, distinct from every Customer and Contact this system
tracks on behalf of the ISP. A Role is the single, fixed authorization
level a User holds.

## Responsibilities

A User records:

-   Email (the unique identity a caller logs in with)
-   Password hash
-   First Name / Last Name (both optional, added 2026-09-10)
-   Role (Administrator, Operator, Viewer)
-   Current status (Active / Inactive)

Role and status answer two different questions, the same "two fields,
two questions" pattern this document already draws elsewhere (see
section 21's Product on Catalog vs. Provider): Role decides what an
active User is allowed to do; status decides whether they may
authenticate at all. Deactivating a User does not change their Role, and
promoting a User does not reactivate them.

First Name/Last Name carry no identity meaning of their own -- nothing
looks a User up by name, and a User can be created and keep signing in
with neither ever set. They exist purely so the system can show *who*
did something (a Note's author, the account menu) as a real name instead
of always falling back to Email, everywhere the display convention is
"First Last, else Email." Unlike Email and Role, a User sets or changes
their own First Name/Last Name (and password) themselves, through a
self-service Profile screen reached from the account menu -- the one
User Management action (section 16) that never requires the
Administrator-only capability every other change to a User does, since a
User is always allowed to edit their own account.

Role is deliberately a single flat enum, not a hierarchy or a set of
composable permissions — RBAC v1 (`internal/authz`) answers
"can this Role do X" with one explicit, statically-typed predicate per
capability, never a generic permission table. A User holds exactly one
Role; there is no multi-role assignment.

A User is never deleted — see the section 17 invariant this adds below.
Nothing here couples a User to a Customer, a Provider, or any inventory
record: authentication identity and the operational entities Palladium
manages are separate concerns, the same separation CLAUDE.md's Core
Philosophy already draws between Customers and Resources.

------------------------------------------------------------------------

# 26. Customer Device

A Customer Device records that a physical Device is placed at a
Customer's premises — installed, sitting there, plugged in — regardless
of whether any Service has been set up to use it yet.

This entity exists because real installs do not always line up with
service activation: a technician can rack an ONT in a customer's home
days before the account goes live, and Palladium needs somewhere to
record "this Device is now at this Customer's address" the moment that
happens, not only once a Service Equipment link exists to imply it.

`internal/customerdevice` is the one deliberate, narrow exception to
this document's stated rule (section 4) that Customers do not directly
own network equipment, and to CLAUDE.md's Core Philosophy ("never couple
inventory directly to customers"). It is scoped as tightly as Service
Equipment already is one domain over (section 7) — a placement record
with a start and end, never a field on Device itself, and never
anything `internal/inventory` (Device's own package) knows about — not a
general "Customer owns Devices" model. A Device placed this way still
carries no billing, provisioning, or service meaning on its own; that
only ever comes from a real Service Equipment assignment.

## Responsibilities

A Customer Device records:

-   Owning Customer
-   Placed Device
-   Location (optional — which of the Customer's Locations the Device
    sits at, for tracking purposes only; added 2026-09-10)
-   Attached date
-   Detached date

Only one active (not-yet-detached) placement should exist for a given
Device at a time — the same rule Service Equipment already enforces for
its own assignments, applied one domain up.

## Interaction with Location

A set Location must belong to the same Customer as the placement itself
— the same "no reaching across Customers" rule this document expects
everywhere a record names more than one other record. This is purely a
"which Location" pointer for an operator glancing at a Customer with
several: it carries no meaning beyond that, the same way this entity's
own Description field once did before it was removed (2026-09-10) for
never being displayed anywhere but a create modal.

`customer_devices.location_id` is `ON DELETE RESTRICT`, and — like every
other field on this entity — a detached Customer Device row is never
deleted, only marked detached (see this section's own opening). The two
facts combined mean a Location can never be hard-deleted once any Device
was ever placed there, even long after that Device was detached and the
placement is pure history: the row enforcing the foreign key is still
sitting there. `internal/location/postgres`'s Delete error names this
specifically ("it still has Device placement history") rather than the
generic foreign-key message it used to give, found live when a
Customer's only Location couldn't be deleted with no Service anywhere on
the page — the real blocker was this, not a Service at all.

## Interaction with Device Status

Placing a Device at a Customer marks it Active (section 6); detaching it
marks it Unused again — but only once no active Service Equipment
assignment also exists for that Device. A Device still fulfilling a live
Service cannot be detached from its Customer at all: the write is
rejected outright (see section 17's corresponding invariant), the same
fail-loud, never-cascade choice this codebase makes throughout rather
than silently tearing down a live Service assignment as a side effect of
a Customer-level action.

## Interaction with Customer Removal

Removing a Customer (section 4's cascade) does not currently detach that
Customer's active Device placements — a known, documented gap as of
2026-09-09, not a decision. A Device can end up still recorded as
placed at an Archived Customer until someone detaches it by hand. Fixing
this means extending the removal cascade (`internal/customer/removal`)
to also detach active Customer Device records, mirroring what it already
does for Service Equipment and Access Attachment.

------------------------------------------------------------------------

# 27. Note

A Note is free-text, operator-authored commentary attached to a
Customer, Device, or Service (added 2026-09-10) — "called the customer
back, issue resolved," "swapped the power adapter," the kind of running
history an operator wants attached to a record that isn't a Service
status change or anything else this document already models as an
Event.

It shares Event's `EntityType`/`EntityID` shape (section 12): a loose
reference, not a typed foreign key, so `internal/note` has no dependency
on `internal/customer`, `internal/inventory`, or `internal/service`, and
nothing stops a future caller from attaching a Note to some other entity
without this domain growing a new field or import for it. The one
deliberate difference from Event is that a Note is always written by a
client — an operator typing into a form — never generated internally by
workflow or domain code the way every Event is. That is also why Note is
its own domain rather than a field or mode added to Event: Event's own
package doc comment states plainly that there is no create route,
because nothing ever posts one from outside the system.

## Responsibilities

A Note records:

-   What it is about (EntityType + EntityID — Customer, Device, or
    Service today)
-   Who wrote it (AuthorUserID, and AuthorEmail captured from that
    caller's authenticated identity at write time)
-   The note text
-   When it was written

AuthorEmail is a deliberate snapshot, not resolved by joining against
the User's current record at read time: a Note keeps showing who wrote
it even if that User's email later changes, and every Role that can read
Notes can see who left one without also needing User Management
permission (section 25) just to resolve a name.

A Note is immutable once written — there is no update or delete anywhere
in this domain, the same "operational history, never changed or
removed" treatment this document already gives Event (see the
Architectural Principle following section 12).

## Interaction with Display

Notes render newest-first, in a collapsible section alongside Timeline
(the display of Event history) on the Customer, Device, and Service
Detail Workspaces — see docs/09-WORKSPACE-SPECIFICATIONS.md sections 8,
9, and 10.

------------------------------------------------------------------------

# 28. Report

A Report is a curated, read-only cross-domain query backing the
Explorer Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md section 15,
added 2026-09-11) — "every customer with their contact info," "every
device," "every customer with the devices tied to them." A Report has no
lifecycle of its own: it is never created, updated, or deleted, and
`internal/report` has no dependency on `internal/customer`,
`internal/inventory`, `internal/contact`, `internal/service`, or
`internal/customerdevice` — each Report row is its own flat,
already-joined shape, not a reuse of any of those domains' own Go types,
even though its SQL reads directly from their tables.

Version 1 ships three Reports, matching exactly what an operator asked
to pull as a CSV rather than a broader set invented ahead of a real
need:

-   **Customers & Contacts** — every Customer, one row per Contact; a
    Customer with none on file still gets exactly one row, with every
    Contact field blank, so a "who do I call" export never silently
    drops a customer.
-   **Devices** — every Device, its resolved Manufacturer/Model, its
    physical Site/Building/Room/Rack path if racked, and whichever
    Customer currently has it, if any.
-   **Customers & Devices** — every currently-active Customer-Device
    relationship, covering both of the two ways section 9's Relationship
    Overview says a Device reaches a Customer: a direct Customer Device
    placement (section 26), or delivery through an active Service
    Equipment assignment (section 7). A Device reaching the same
    Customer both ways at once produces two rows, one per relationship —
    this Report never collapses them into one, unlike the Devices
    Report's single "whichever wins" column above.

Explorer is deliberately a curated set of real, hand-written queries,
not a dynamic ad hoc query builder an operator assembles from arbitrary
filters and joins — a decision made explicitly with the user before
building it, matching how every other domain in this document already
exposes purpose-built reads rather than a generic query layer, and
avoiding a large, novel piece of query-engine infrastructure nothing
else in the codebase needed. A Report's rows still open into its
subject's own Workspace when selected (a Customer row into the Customer
Detail Workspace, a Device row into the Device Detail Workspace) —
Explorer is a way into the rest of Palladium, not a reporting silo
alongside it.

## Responsibilities

`internal/report` exists to answer read queries only:

-   List every row of a named Report (no filter, sort, or pagination
    server-side — see the interaction note below)

Every capability that guards a Report route (`RequireReports`) is read
only, open to every Role (Administrator, Operator, Viewer): a Report
only ever surfaces data each of those Roles can already see
individually, through Customers, Devices, or Contacts.

## Interaction with Display

Unlike every other Collection View in this document, a Report is
fetched once, in full, when selected in Explorer, not re-fetched on
every search/sort/page change — search, sort, pagination, and CSV export
all happen client-side against that one cached array. A Report is a
pull-once CSV-prep tool, not a live filtered browse list.

------------------------------------------------------------------------

# Closing Statement

The Domain Model defines the language of Palladium.

Every feature, workflow, database schema, API, and user interface should
be built upon the concepts described in this document.

By protecting the integrity of the domain model, Palladium remains
understandable, extensible, and maintainable as it grows.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft
  1.1 Draft   2026-09-04   Corrected section 7 and the section 17 invariant: Service Equipment removal is soft by default, but a real hard delete now exists for disposable test/demo data (blocked by an active Access Attachment)
  1.2 Draft   2026-09-05   Added sections 20-24 (Product Catalog, Product, Service Profile, Provider, Provisioning Profile), documenting four real, already-implemented domains this document had never covered; corrected two stale "section 5" citations elsewhere that meant to point at Product
  1.3 Draft   2026-09-07   Added section 25 (User & Role) and a User row in section 16 (Entity Lifecycles), documenting the already-implemented `internal/auth` domain; added the corresponding section 17 invariant (Users are never deleted)
  1.4 Draft   2026-09-09   Added section 26 (Customer Device), documenting `internal/customerdevice` -- the one deliberate exception to section 4's "Customers do not directly own network equipment" rule, letting a Device be placed at a Customer's premises before any Service exists. Corrected section 6 and the section 16 Device row: Device status is a flat Unused/Active/Retired set (not the original seven-value procurement progression, which never matched Device's actual CPE-only scope), Active now derives from either a Service Equipment assignment or a Customer Device placement, and a Device is never permanently deleted (removed the corresponding stale hard-delete assumption). Added section 7 coverage of `internal/onuauthorization`'s fallback OLT-resolution role, and new section 17 invariants for Customer Device's own one-active-placement rule and its detach-blocked-while-in-service rule
  1.5 Draft   2026-09-10   Added section 7 coverage of the new Access Attachment auto-sync: authorizing a Device via the OLT blacklist flow now also finds-or-creates its PON Port/Access Interface (`internal/provisioning/kontron/service`), and creating its Service Equipment record auto-creates the matching Access Attachment (`ServiceEquipmentService.syncAccessAttachment`) when both are on file -- closing the gap flagged in 1.4's docs-sync pass where this was manual-only
  1.6 Draft   2026-09-10   Added section 27 (Note), documenting the new `internal/note` domain. Added a Location field to section 26 (Customer Device) and an "Interaction with Location" subsection covering why a Location with any Device placement history can never be hard-deleted; removed section 26's Description field, and the stale Description bullet from sections 20-24 (Product Catalog, Product, Service Profile, Provider, Provisioning Profile) -- Description was removed from all ten domains for never displaying anywhere but a create modal (CustomerType's "Government" value was removed for the same reason: modeled but never used). Added section 17 invariants for Note immutability and the Location-delete restriction
  1.7 Draft   2026-09-11   Added optional First Name/Last Name to section 25 (User & Role), and documented that a User edits their own name/password through a self-service Profile screen, not the Administrator-only User Management flow. Added section 28 (Report), documenting the new `internal/report` domain backing Explorer (docs/09-WORKSPACE-SPECIFICATIONS.md section 15): three curated cross-domain reports (Customers & Contacts, Devices, Customers & Devices), not a dynamic ad hoc query builder
  1.8 Draft   2026-09-11   Removed Access Network as a domain concept: Palladium only ever manages a single physical network per instance, so requiring every OLT to belong to a named Access Network grouping was an unused layer of indirection nothing else in the domain model ever referenced. OLT is now the root of the Network hierarchy (OLT → PON Port → Access Interface → Access Attachment). Corrected the Device, Site, and Relationship Overview sections' "Access Network" callouts accordingly
  1.9 Draft   2026-09-11   Corrected section 12 (Event): most of its example Events are now real (Customer/Device/Contact/Location created, Device retired, Service added/removed, Service Equipment attached/detached), each written directly from its own domain's httpapi handler rather than through a Workflow -- corrected the Architectural Principle's "Workflows create Events" to describe Workflow's own relationship to Event specifically, not an exclusivity claim. ONU discovered and Router synchronized remain aspirational

------------------------------------------------------------------------

# Related Documents

-   01-VISION.md
-   02-DESIGN-PRINCIPLES.md
-   04-NAVIGATION.md
-   05-WORKFLOW-ENGINE.md

------------------------------------------------------------------------

**End of Document**