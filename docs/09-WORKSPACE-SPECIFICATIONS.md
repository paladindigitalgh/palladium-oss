---
document: 09-WORKSPACE-SPECIFICATIONS
status: Draft
title: Workspace Specifications
version: 1.4-draft
---

# Workspace Specifications

## Executive Summary

This document defines the functional specification for every primary
Workspace within Palladium.

While the UI Architecture describes *how* the interface is structured,
this document specifies *what each workspace contains*, *what operators
can do there*, and *how related information is organized*.

These specifications serve as the implementation blueprint for the Vue
frontend.

------------------------------------------------------------------------

# Table of Contents

1.  Purpose
2.  Design Goals
3.  Workspace Standards
4.  Workspace Archetypes
5.  Collection View & Detail View
6.  Detail Workspace Structure
7.  Dashboard Workspace
8.  Customer Workspace
9.  Service Workspace
10. Device Workspace
11. Network Workspace (Access Network / OLT / PON Port / Access Interface)
12. Site Workspace
13. Workflow Workspace
14. Search Results Workspace
15. Explorer Workspace
16. Administration Workspace
17. Global Workspace Behaviors
18. Cross-Workspace Navigation
19. Workspace Permissions
20. Future Workspaces

------------------------------------------------------------------------

# 1. Purpose

Every Workspace should provide a complete operational view of a single
entity or task.

Operators should rarely need to leave a workspace to complete common
activities.

This document establishes a consistent structure for every workspace in
the application, appropriate to its archetype (see section 4, "Workspace
Archetypes").

------------------------------------------------------------------------

# 2. Design Goals

Every workspace should:

-   Present the most important information first
-   Expose common actions prominently
-   Surface related resources naturally
-   Preserve operator context
-   Support keyboard-first navigation
-   Integrate seamlessly with the Workflow Engine

"Surface related resources naturally" applies where a workspace has
related resources to surface -- the Landing Workspace has none (see
section 4, "Workspace Archetypes").

------------------------------------------------------------------------

# 3. Workspace Standards

Every workspace should define:

-   Purpose
-   Primary audience
-   Header content
-   Primary actions
-   Panels
-   Related resources
-   Available workflows
-   Navigation behavior
-   Empty states

These standards ensure a consistent operator experience.

"Related resources" and "Available workflows" apply to Entity
Workspaces (see section 4, "Workspace Archetypes"). The Landing
Workspace has no selected entity, so it has neither. For a Detail
Workspace, "Panels" means its **sections** (section 6, "Detail
Workspace Structure").

------------------------------------------------------------------------

# 4. Workspace Archetypes

Not every Workspace manages an entity. Palladium OSS distinguishes two
Workspace archetypes, and every Workspace is exactly one of them.

## Landing Workspaces

**Purpose:** provide a high-level operational overview rather than
managing a specific entity.

**Characteristics:**

-   Dashboard-oriented
-   Metric cards
-   Operational widgets
-   Status summaries
-   No selected entity
-   No Relationship Panel
-   No Timeline Panel
-   Do not use WorkspaceLayout (docs/11-COMPONENT-ARCHITECTURE.md)

Version 1 has exactly one Landing Workspace: **Dashboard** (section 7).

## Entity Workspaces

**Purpose:** manage and interact with one or more OSS entities.

**Characteristics:**

-   Lists
-   Filters
-   Detail views
-   Actions
-   Relationships
-   Timeline
-   Consistent workspace structure (docs/11-COMPONENT-ARCHITECTURE.md,
    "Workspace Architecture")
-   Use WorkspaceLayout

Version 1's primary-navigation Entity Workspaces are **Customers,
Services, Devices, Network, Inventory, Explorer, Administration**.

Every other workspace specified later in this document -- Customer,
Service, Device, OLT, Site, Workflow, and Search Results -- is also an
Entity Workspace, whether it is reached from primary navigation, by
drilling into a related resource, or from search. Being reachable from
primary navigation is not what makes a workspace an Entity Workspace;
managing an entity is. Dashboard is the only workspace that does not.

Every Entity Workspace is composed of two primary views: a **Collection
View** and a **Detail View**. These are not additional Workspace
archetypes -- Landing Workspace and Entity Workspace remain the only two
-- they are how an Entity Workspace is internally structured. See section
5, "Collection View & Detail View."

------------------------------------------------------------------------

# 5. Collection View & Detail View

Primary Navigation leads to a Collection View, which leads to a Detail
View:

``` text
Primary Navigation
        |
        v
Collection View
        |
        v
Detail View
```

This is a navigation *flow* within a single Entity Workspace, not three
different kinds of Workspace. Discovery Before Detail
(docs/02-DESIGN-PRINCIPLES.md, principle 5) is the design principle this
section specifies in functional detail.

## Collection View

**Purpose:**

-   Browse objects
-   Search
-   Filter
-   Sort
-   Select an object

Collection Views should:

-   Display only identifying information
-   Support search
-   Support filtering
-   Support sorting
-   Navigate to the object's Detail View

Collection Views should **not**:

-   Become dashboards
-   Display operational metrics
-   Duplicate Detail View functionality
-   Present large amounts of object-specific information

The exact columns vary by entity, but every Collection View should
remain intentionally minimal. For example:

**Customers:** Customer, Location, Status

**Devices:** Device (name + serial number), Manufacturer / Model, Status, Created

**Inventory:** Asset, Model, Location

## Detail View

The Detail View is the canonical operational interface for an object --
where technicians perform work. Typical responsibilities include:

-   Complete object information
-   Relationships
-   Timeline
-   Activity
-   Operational actions
-   Configuration
-   Related resources

The Detail View is realized as a **Detail Workspace**: a single
continuous page composed of collapsible sections, not multiple pages,
tabs, or nested navigation. See section 6, "Detail Workspace Structure,"
for the full specification, and docs/02-DESIGN-PRINCIPLES.md, principle
6, "Single-Workspace Operations," for the design principle it
implements. "Detail View" describes *where navigation leads*; "Detail
Workspace" describes *what is there once you arrive* -- the same
concept, named precisely for each concern.

## Navigation Flow

Clicking a primary navigation item opens that Entity Workspace's
Collection View.

-   Customers -> Customer Collection View -> Customer Detail View
-   Devices -> Device Collection View -> Device Detail View
-   Services -> Service Collection View -> Service Detail View
-   Inventory -> Inventory Collection View -> Inventory Asset Detail View
-   Network -> Network Collection View -> appropriate operational Detail
    View (OLT, Site, etc.)

Explorer remains a special case: it is a query engine that links into
existing Detail Views rather than owning its own operational entities,
so it has no Collection View of its own in the same sense.

## Canonical Detail Views

A primary operational object should have exactly one canonical Detail
View, and it should open into that same Detail View regardless of how it
was reached.

For example, a Customer opened from Search, Explorer, a Device, a
Service, or an Alert should always open the same Customer Detail View.

This creates a predictable user experience: an object's Detail View is
never reimplemented per entry point.

This does not mean every object requires a Detail View. Configuration
objects and simple reference data may continue to use inline editing
where appropriate.

------------------------------------------------------------------------

# 6. Detail Workspace Structure

A Detail Workspace is not divided into multiple pages, tabs, or nested
navigation. It is a single continuous operational workspace composed of
sections, each containing one category of information related to the
current object. The entire workspace is visible by default: technicians
scroll naturally through the page, or jump directly to a section using
the Contents navigation.

This is docs/02-DESIGN-PRINCIPLES.md principle 6, "Single-Workspace
Operations," specified in functional detail. This pattern is reused
consistently across the Customer, Device, Service, Inventory, Network,
and future Detail Workspaces (see section 4, "Workspace Archetypes," for
which workspaces those are).

Example (Customer Workspace): Summary, Locations, Services, Timeline
(see section 8 for the current, real section list).

## Workspace Header

Displays:

-   Object identity
-   Operational status
-   Primary identifying information
-   Primary actions

## Contents Navigation

A persistent in-page navigation listing every section.

Responsibilities:

-   Jump to sections
-   Highlight the currently visible section
-   Support smooth scrolling
-   Remain visible while scrolling when practical

This is in-page navigation only. It is **not** another layer of
application navigation -- it does not change the URL's Detail Workspace,
only the scroll position within it.

## Sections

Each section should:

-   Represent one operational category
-   Be independently collapsible
-   Be expanded by default
-   Remain part of one continuous page
-   Avoid nested navigation

Examples: Summary, Services, Devices, Timeline, Notes. The exact
sections vary by workspace.

Collapse state may be remembered as a user preference (see
docs/02-DESIGN-PRINCIPLES.md, principle 6).

## Tabs

Detail Workspaces intentionally avoid tabs and nested sub-pages.

Operators should not have to search multiple tabs to understand the
current object. Scrolling is preferred over navigation.

## Component Architecture

See docs/11-COMPONENT-ARCHITECTURE.md, "Workspace Architecture," for the
reusable component hierarchy (DetailWorkspace, WorkspaceHeader,
ContentsNavigation, SectionContainer, and per-entity Section components)
and docs/08-DESIGN-SYSTEM.md for the `SectionCard` component every
section is built from.

------------------------------------------------------------------------

# 7. Dashboard Workspace

**Archetype:** Landing Workspace (section 4). Dashboard does not use
WorkspaceLayout.

## Purpose

The Dashboard is the operator's landing page and operational overview.

It answers one question:

**"What needs my attention right now?"**

## Primary Audience

-   Network Operations
-   Support Technicians
-   Administrators

## Header

`WorkspaceHeader` with a title and subtitle only — no organization
selector, global search, current time, active-workflow indicator, or
notification center. None of those exist anywhere in this app yet (no
multi-tenant concept, no global search, no notifications); the header
does not simulate having them.

## Stats

Four `StatisticCard`s, each a real query, not a mock number:

-   Customers — total count.
-   Active Services — count with status `Active`.
-   Devices — total count.
-   Pending Tasks — count of `WorkflowInstance`s with status `Pending`
    or `Failed`. `warning` variant when non-zero, `neutral` when zero
    (docs/08-DESIGN-SYSTEM.md section 3: calm under normal operation,
    visually emphatic only when action is required).

A "System Health" stat was considered and deliberately left out:
`/healthz`/`/readyz` exist but are mounted outside `/api/v1`
(unauthenticated, a different base path than every other endpoint this
frontend calls), and wiring one stat to a separate fetch mechanism was
not worth what it would buy.

## Widgets

Three `DashboardWidget`s, each backed by `useDashboard.ts`:

-   **Recent Activity** — the most recent Events system-wide (`GET
    /api/v1/events/recent`, a small, deliberately bounded addition to
    the Event domain — see `internal/event/httpapi`'s own doc comment
    for why this is a different, safe shape from the per-entity
    `/events` endpoint, which stays unbounded-refused by design).
-   **Network Overview** — real counts: Access Networks, OLTs, PON
    Ports, and Access Interfaces split by Active/Disabled (a real
    administrative field, not live telemetry). No online/active ratios
    — Palladium has no telemetry to report one.
-   **Pending Tasks** — the actual Pending/Failed `WorkflowInstance`s
    behind the stat above, each linking to its Service.

"Active Alerts" (a stat and a widget in earlier drafts of this
document) does not exist and will not: no alert concept exists
anywhere in the domain model, and per CLAUDE.md, Palladium is not a
monitoring platform. Same reasoning that already excludes
Alarms/Performance sections from the Device and Service Detail
Workspaces.

## Primary Actions

None today — the Dashboard is read-only. `Create Customer` /
`Provision Service` / etc. live on their own workspaces, reached via
global navigation, not launched from here.

## Design Principle

The Dashboard should summarize operations, not replace dedicated
workspaces. Every panel above is real data or absent — nothing on this
page is illustrative.

# 8. Customer Workspace

## Purpose

The Customer Workspace is the central location for viewing and managing
a customer relationship.

It should answer:

**"Who is this customer, what locations and services do they have, and
what has happened recently?"**

## Primary Audience

-   Customer Support
-   Network Operations
-   Provisioning
-   Billing Integration (future)

## Header

Display:

-   Customer name
-   Customer type (Residential, Business, Government, Internal), as the
    subtitle
-   Status
-   Customer ID, as metadata

A Customer is an identity record only (docs/03-DOMAIN-MODEL.md, section
4): no account number, tags, or alerts exist on the Customer record
itself, so none of these appear in the header. Contact information is
real (docs/03-DOMAIN-MODEL.md, section 19) but is a related entity
resolved on demand, the same as Locations, not a header field.

## Primary Actions

-   Edit Customer
-   Delete Customer
-   Add Contact
-   Add Location
-   Add Service

Provisioning and suspending a service, and every other workflow-driven
action, live on the **Service** Workspace (section 9), not here -- a
Customer can have several Services, each independently provisioned,
suspended, or resumed.

## Sections

-   Summary (status, customer type, created date, description)
-   Contacts -- every Contact on the account, with Add/Edit/Remove
-   Locations -- every Location on the account, with Add/Edit/Remove
-   Services -- every Service across every Location, with Add/Remove;
    opens the Service Workspace
-   Timeline -- the Customer's real audit trail (docs/02-DESIGN-PRINCIPLES.md
    principle 10), sourced from the Event domain

Equipment, workflow history, and diagnostics belong to a Service, not a
Customer directly (docs/03-DOMAIN-MODEL.md: a Customer owns Services
through Locations, and equipment is associated through Services) -- see
those sections on the Service Workspace instead. There is no Notes
feature in Version 1.

## Navigation

Every related location or service should open its own Workspace while
preserving the Customer Workspace.

------------------------------------------------------------------------

# 9. Service Workspace

## Purpose

The Service Workspace focuses on a single customer service. It is where
the Workflow Engine is actually exercised: provisioning, suspending, and
resuming a Service each run a real Workflow (05-WORKFLOW-ENGINE.md)
against the vendor plugin registered for it.

It should answer:

**"Is this service correctly provisioned, and what has it done?"**

## Header

Include:

-   Service identifier
-   Status

A Service record (docs/03-DOMAIN-MODEL.md) is lean: no technology,
provisioned speed, or utilization field exists on it. Customer and
Location are shown as their own sections below, not header fields --
richer service detail (technology, speed tier) belongs to Product and
Service Profile, neither of which has its own read model yet.

## Primary Actions

-   Edit Service
-   Delete Service
-   One dynamic primary action, following the Service's current status:
    **Provision Service** (Pending), **Suspend Service** (Active), or
    **Resume Service** (Suspended). Each runs the matching Workflow
    Definition (provision-service, suspend-service, resume-service --
    see 05-WORKFLOW-ENGINE.md) and updates the Service's own status when
    it succeeds.

There is no speed-profile change, diagnostics, or configuration-view
action in Version 1.

## Sections

-   Summary (status, activated/suspended/disconnected dates, description)
-   Customer -- the owning Customer, resolved through the Service's
    Location
-   Location -- the Service's Location
-   Equipment -- the Service's Service Equipment assignments (role,
    device, installed date)
-   Workflow History -- every WorkflowInstance run against this Service
    (definition, status, started date)
-   Timeline -- the Service's real audit trail, sourced from the Event
    domain

There are no Performance or Active Alarms sections -- that data is
monitoring/telemetry, out of scope per CLAUDE.md ("Palladium is NOT... a
monitoring platform").

------------------------------------------------------------------------

# 10. Device Workspace

## Purpose

The Device Workspace provides a complete inventory view of an individual
physical Device: what it is, where it sits in the Rack hierarchy
(docs/03-DOMAIN-MODEL.md), its lifecycle status, and which Service, if
any, it currently fulfills.

Device is deliberately generic -- there is no type/subtype field, no
"OLT" or "ONU" or "Router" distinction (docs/03-DOMAIN-MODEL.md, section
6). This is not a gap to fill in later: a Device workspace showing
software version, interfaces, performance, or alarms would be a
monitoring/NMS feature, and CLAUDE.md is explicit that Palladium is NOT a
monitoring platform ("Monitoring belongs in Zabbix or other monitoring
systems"). This workspace's scope stops at inventory and assignment.

## Header

Display:

-   Device name
-   Manufacturer and model, as the subtitle
-   Status (Ordered, Received, In Stock, Installed, Maintenance,
    Retired, or Disposed -- docs/03-DOMAIN-MODEL.md, section 16)
-   Serial number, and asset tag when set, as metadata

## Primary Actions

-   Edit Device
-   Delete Device

## Sections

-   Summary (manufacturer, model, asset tag, created/updated dates,
    description)
-   Assignment -- every active Service Equipment link, each resolving to
    the Service it fulfills; an empty state when the Device is not
    currently assigned
-   Timeline -- the Device's real audit trail, sourced from the Event
    domain

There are no Interfaces, Configuration, Performance, Alarms, or Running
Workflows sections -- see this section's Purpose note above.

------------------------------------------------------------------------

# Design Principle

Customer, Service, and Device Workspaces should feel related while
presenting information appropriate to their specific purpose.

# 11. Network Workspace (Access Network / OLT / PON Port / Access Interface)

## Purpose

The Network Workspace is a four-level operational hierarchy — Access
Network → OLT → PON Port → Access Interface — reached from the Network
Collection View (a list of Access Networks). Each level is its own
canonical Detail View, not a tab or a page nested under its parent:
opening an OLT navigates to `/network/olts/{id}`, exactly as opening a
Customer navigates to `/customers/{id}` (section 5's canonical-URL
principle applies here too). A fifth level, Access Attachment (an
equipment item attached to an Access Interface), has no Detail View of
its own — it is managed inline on the Access Interface Workspace,
mirroring how Service Equipment has no Detail View of its own either.

It should answer, at each level: **"What exists under this node, and
what is it connected to above it?"**

## Access Network Workspace

### Header

-   Access Network name
-   Status (Active / Inactive)

### Primary Actions

-   Edit Access Network
-   Delete Access Network

### Sections

-   Summary (status, created)
-   OLTs (add, remove, open)
-   Timeline

## OLT Workspace

### Header

-   OLT name
-   Subtitle: vendor
-   Management IP, when set

### Primary Actions

-   Edit OLT
-   Delete OLT

### Sections

-   Summary (vendor, model, management IP, created)
-   Access Network (a single relationship link back up the chain)
-   PON Ports (add, remove, open)
-   Timeline

There is no OLT-level status, software version, uptime, health, or
alarm data here — an OLT record answers "what is this device and which
Access Network does it belong to," not "is it currently healthy." That
question belongs to a monitoring system (Zabbix, LibreNMS, Prometheus),
per CLAUDE.md: Palladium is not a monitoring platform, and this
workspace does not simulate being one.

## PON Port Workspace

The thinnest workspace in the app — a PON port has exactly one
meaningful field of its own (its port number) and exists mainly to
connect an OLT to its Access Interfaces.

### Header

-   Title: "Port {portNumber}"

### Primary Actions

-   Edit PON Port
-   Delete PON Port

### Sections

-   Summary (created, last updated)
-   OLT (a single relationship link back up the chain)
-   Access Interfaces (add, remove, open)
-   Timeline

## Access Interface Workspace

### Header

-   Access Interface name
-   Subtitle: technology (GPON / XGS-PON / Active Ethernet / Other)
-   Status (Active / Disabled)

### Primary Actions

-   Edit Access Interface
-   Delete Access Interface

### Sections

-   Summary (technology, status, created)
-   PON Port (a single relationship link back up the chain)
-   Attachments — every Access Attachment ever made to this interface,
    active and removed alike; removal is history, not deletion (see
    03-DOMAIN-MODEL.md). "Attach Equipment" opens a picker over
    existing Service Equipment; "Detach" records a reason and a
    removal timestamp rather than deleting the row.
-   Timeline

------------------------------------------------------------------------

# 12. Site Workspace

## Purpose

The Site Workspace presents all operational resources associated with a
physical location.

It should answer:

**"What equipment, services, and issues exist at this site?"**

## Header

Display:

-   Site name
-   Address or location
-   Site type
-   Operational status
-   Contact information (where applicable)

## Primary Actions

-   View inventory
-   Open topology
-   Run site diagnostics
-   View maintenance history

## Sections

-   Site Summary
-   Installed Equipment
-   Network Topology
-   Active Services
-   Environmental Alerts
-   Timeline

------------------------------------------------------------------------

# 13. Workflow Workspace

## Purpose

The Workflow Workspace provides complete visibility into a workflow
execution.

It should answer:

**"What is this workflow doing, what has already happened, and what
happens next?"**

## Header

Display:

-   Workflow name
-   Status
-   Initiating operator
-   Start time
-   Duration
-   Target resource

## Primary Actions

-   Cancel (if permitted)
-   Retry failed workflow
-   View generated events
-   Open related resources

## Sections

-   Workflow Summary
-   Step Progress
-   Execution Log
-   Generated Events
-   Related Resources
-   Timeline

------------------------------------------------------------------------

# 14. Search Results Workspace

## Purpose

Search Results provide a unified view of everything matching the
operator's query.

Results may include:

-   Customers
-   Services
-   Devices
-   Sites
-   Workflows
-   Events

## Primary Actions

-   Filter results
-   Save search
-   Open selected Workspace
-   Bulk actions (where appropriate)

## Primary Panels

Search Results has no single object, so it is not a Detail Workspace and
does not use section 6, "Detail Workspace Structure" -- selecting a
result opens that object's own Detail Workspace instead.

-   Result List
-   Filters
-   Recent Searches
-   Saved Searches

------------------------------------------------------------------------

# Design Principle

Infrastructure workspaces should expose relationships between resources
so operators can move naturally from high-level health to detailed
investigation.

# 15. Explorer Workspace

## Purpose

Explorer is Palladium's ad hoc query and reporting engine. It lets
operators explore relationships within the OSS database, review results
directly in the interface, and export them for further use.

It should answer:

**"Which records match a specific operational condition, right now?"**

Explorer is not a topology viewer, map, or visualization tool. It has no
notion of physical or logical network diagrams; its subject is data, not
diagrams. Network topology is addressed by dedicated workspaces (see
Site Workspace, section 12, and the future Network Topology workspace,
section 20).

## Primary Audience

-   Network Operations
-   Customer Support
-   Provisioning
-   System Administrators

## Primary Actions

-   Build or select a query
-   Run a query across one or more domains
-   Filter and sort results
-   Open a result directly into its own Workspace
-   Export results to CSV
-   Save a query for reuse (future)

## Primary Panels

Explorer is a query engine, not a single-object workspace, so it is not
a Detail Workspace and does not use section 6, "Detail Workspace
Structure" -- opening a result leads to that object's own Detail
Workspace instead.

-   Query Builder
-   Result Table
-   Saved Queries (future)
-   Export Options

## Example Queries

-   Show all customers on a specific OLT
-   Show all customers without a service
-   Show all ONUs that have never informed
-   Show all devices with outdated firmware
-   Show all inventory assigned to a specific site
-   Show all subscribers on a VLAN

------------------------------------------------------------------------

# Design Principle

Explorer trades the fixed structure of a traditional reporting page for
the flexibility of a direct question. Every result should open into its
subject's own Workspace, keeping Explorer consistent with the rest of
Palladium rather than a separate reporting silo.

# 16. Administration Workspace

## Purpose

The Administration Workspace centralizes platform configuration and
operational administration.

It should answer:

**"How is Palladium configured, and what administrative tasks require
attention?"**

## Primary Audience

-   System Administrators
-   Network Engineers
-   Platform Operators

## Primary Actions

-   Manage users and roles
-   Configure plugins
-   View system health
-   Manage secrets
-   Configure integrations
-   Review audit logs

## Primary Panels

Administration centralizes independent platform-configuration concerns
rather than managing one object, so it is not a Detail Workspace and
does not use section 6, "Detail Workspace Structure."

-   System Health
-   User Management
-   Roles & Permissions
-   Plugin Management
-   Integrations
-   Audit Log
-   Platform Settings

------------------------------------------------------------------------

# 17. Global Workspace Behaviors

All workspaces should behave consistently.

Common behaviors include:

-   Breadcrumb navigation
-   Keyboard shortcuts
-   Saved state
-   Deep linking
-   Timeline access
-   Related resource navigation
-   Workflow launch from context

Operators should not need to relearn interactions when changing
workspaces.

------------------------------------------------------------------------

# 18. Cross-Workspace Navigation

Relationships are first-class navigation elements.

Examples include:

-   Customer → Service
-   Service → ONU
-   ONU → OLT
-   OLT → Site
-   Workflow → Target Resource
-   Event → Related Entity

Navigation should preserve existing workspaces while making it easy to
follow relationships.

------------------------------------------------------------------------

# 19. Workspace Permissions

Visibility and actions are permission-aware.

Permissions determine:

-   Workspace visibility
-   Editable fields
-   Available workflows
-   Administrative actions
-   Sensitive information access

The interface should hide unavailable actions rather than presenting
unusable controls.

------------------------------------------------------------------------

# 20. Future Workspaces

Future versions may introduce dedicated workspaces for:

-   Events
-   Inventory
-   Network Topology
-   Maintenance Windows
-   Capacity Planning
-   AI Operations

Ad hoc querying and reporting is not on this list: Explorer (section 15)
already provides it as a Version 1 workspace, not a future one.

These should follow the same structural standards defined in this
document.

------------------------------------------------------------------------

# Closing Statement

Workspaces are the primary way operators interact with Palladium.

By organizing information around operational entities instead of
isolated pages, each workspace becomes a complete environment for
understanding, investigating, and acting on the network.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft
  1.1 Draft   2026-07-30   Added Explorer Workspace (section 12) as the OSS query and reporting engine; removed Reporting from Future Workspaces
  1.2 Draft   2026-07-30   Documented the Landing/Entity Workspace archetype distinction (section 4); clarified WorkspaceLayout applies to Entity Workspaces only
  1.3 Draft   2026-07-30   Added Collection View & Detail View specification (section 5): Collection View discovery scope, Detail View responsibilities, navigation flow, and canonical Detail Views
  1.4 Draft   2026-07-30   Added Detail Workspace Structure (section 6): header, Contents navigation, collapsible sections, no tabs; renamed "Primary Panels" to "Sections" for single-object Detail Workspaces

------------------------------------------------------------------------

# Related Documents

-   02-DESIGN-PRINCIPLES.md
-   04-NAVIGATION.md
-   05-WORKFLOW-ENGINE.md
-   07-UI-ARCHITECTURE.md
-   08-DESIGN-SYSTEM.md
-   11-COMPONENT-ARCHITECTURE.md

**End of Document**
