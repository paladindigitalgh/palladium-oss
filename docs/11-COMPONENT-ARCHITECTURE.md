# 11 - Component Architecture

**Status:** Draft\
**Audience:** Frontend developers, plugin developers, architects

------------------------------------------------------------------------

# Purpose

This document defines the reusable component architecture for the
Palladium OSS frontend.

The goals are to:

-   Promote consistency across the application.
-   Maximize component reuse.
-   Support long-term maintainability.
-   Enable plugin extensibility without modifying core UI.
-   Provide predictable layouts for operators.
-   Keep business logic separate from presentation.

This document complements the UI Architecture and Navigation documents
by defining **how the UI is constructed**, rather than **what screens
exist**.

------------------------------------------------------------------------

# Design Principles

## Composition over inheritance

Vue components should be composed from smaller reusable components
rather than extending one another.

## Separate business logic from presentation

Business logic belongs in:

-   Composables
-   Services
-   API clients
-   Stores

Presentation components should primarily receive data via props and emit
events.

## Consistent workspace experience

Every major object (Customer, Device, OLT, ONU, Service, Workflow,
Plugin, Site, etc.) should feel structurally identical. Concretely: each
has one canonical Detail View that opens the same way regardless of
where it was reached from (docs/09-WORKSPACE-SPECIFICATIONS.md,
"Canonical Detail Views").

Operators should learn one interface rather than dozens.

## Reuse before creating

Before introducing a new component, determine whether an existing shared
component can satisfy the requirement.

## Plugins extend, never replace

Plugins should integrate through documented extension points instead of
replacing or modifying core UI components.

------------------------------------------------------------------------

# High-Level Component Hierarchy

``` text
App
└── AppShell
    ├── Sidebar
    ├── TopNavigation
    ├── GlobalSearch
    ├── NotificationCenter
    ├── WorkflowDrawer
    ├── CommandPalette
    └── WorkspaceHost
```

The AppShell should remain persistent throughout the lifetime of the
application. Only the WorkspaceHost changes as users navigate.

------------------------------------------------------------------------

# Workspace Archetypes

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
-   Do not use WorkspaceLayout

Version 1 has exactly one Landing Workspace: **Dashboard**.

## Entity Workspaces

**Purpose:** manage and interact with one or more OSS entities.

**Characteristics:**

-   Lists
-   Filters
-   Detail views
-   Actions
-   Relationships
-   Timeline
-   Consistent workspace structure (see "Workspace Architecture" below)
-   Use WorkspaceLayout

Version 1's primary-navigation Entity Workspaces are: **Customers,
Services, Devices, Network, Inventory, Explorer, Administration**.

Every other workspace specified in
docs/09-WORKSPACE-SPECIFICATIONS.md -- for example Customer, Service,
Device, OLT, Site, Workflow, and Search Results -- is also an Entity
Workspace, whether it is reached from primary navigation, by drilling
into a related resource, or from search. "Landing Workspace" describes
Dashboard's purpose, not simply "not in the sidebar": a workspace reached
by drilling into an entity is still managing an entity, so it is an
Entity Workspace regardless of how it was opened.

Every Entity Workspace is composed of two primary views: a **Collection
View** and a **Detail View** (docs/09-WORKSPACE-SPECIFICATIONS.md,
"Collection View & Detail View"). This is not a third Workspace
archetype -- Landing Workspace and Entity Workspace remain the only two
-- it is how an Entity Workspace is internally structured: Primary
Navigation opens the Collection View; selecting an object opens its
Detail View.

------------------------------------------------------------------------

# Workspace Architecture

Every **Entity Workspace** (see "Workspace Archetypes" above) should
follow the same structure for its **Detail Workspace**
(docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure").
**Landing Workspaces are deliberately exempt** from this structure: with
no selected entity, they have no sections to show, and Dashboard's own
layout (summary cards plus operational widgets) does not fit this shape
at all.

The tree below is the Detail Workspace's structure specifically. An
Entity Workspace's Collection View is a different, simpler shape --
typically a DataTable plus search/filter/sort controls (see "Shared
Components," Data Display) -- not this tree; see
docs/09-WORKSPACE-SPECIFICATIONS.md, "Collection View & Detail View,"
for what a Collection View should and should not contain.

``` text
DetailWorkspace
├── WorkspaceHeader
├── ContentsNavigation
└── SectionContainer
    ├── SummarySection
    ├── ServicesSection
    ├── DevicesSection
    ├── TimelineSection
    └── ... (additional Section components, one per operational
            category; the exact set varies by workspace)
```

This tree is a **single continuous page**, not a page-plus-sidebar
layout: `ContentsNavigation` is in-page navigation for jumping between
sections, not a second content region running alongside
`SectionContainer` (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail
Workspace Structure," section 6, "Contents Navigation" -- "This is
in-page navigation only. It is not another layer of application
navigation"). Every section under `SectionContainer` is built from the
same `SectionCard` component (docs/08-DESIGN-SYSTEM.md), independently
collapsible and expanded by default.

## Workspace Header

Displays:

-   Object identity
-   Operational status
-   Primary identifying information
-   Primary actions

## Contents Navigation

A persistent in-page navigation listing every section: jumps to a
section, highlights the currently visible one, supports smooth
scrolling, and remains visible while scrolling when practical. See
docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure," for
the full specification.

## Section Container and Sections

Each section under `SectionContainer` represents one operational
category (docs/02-DESIGN-PRINCIPLES.md, principle 6, "Single-Workspace
Operations") -- for example, on a Customer Workspace: Summary, Services,
Devices, Contacts, Alerts, Activity, Timeline, Notes.

What earlier revisions of this document called `WorkspaceSummary`,
`WorkspaceContent`, `RelationshipPanel`, and `TimelinePanel` are now
understood as **Sections**, not separate panel types in a sidebar:
`WorkspaceSummary` becomes a `SummarySection`; `WorkspaceContent`'s
domain-specific information becomes one or more named sections (e.g.
`ProvisioningDetailsSection`); what `RelationshipPanel` displayed is now
shown by the specific relationship a section names (a Customer
Workspace's `ServicesSection` and `DevicesSection` *are* its
relationships, rather than one generic relationship panel); and
`TimelinePanel` becomes `TimelineSection`. Relationships remain always
navigable -- a row within `ServicesSection` or `DevicesSection` still
opens that object's own Detail Workspace
(docs/09-WORKSPACE-SPECIFICATIONS.md, "Canonical Detail Views").

`InspectorPanel (optional)`, previously listed as a sibling of
`RelationshipPanel`/`TimelinePanel`, has no equivalent in this structure;
an inspector-style need is expected to be met by a Section like any
other.

## Timeline Section

Chronological history including:

-   Provisioning
-   Configuration
-   Audit events
-   Inventory changes
-   Plugin actions

------------------------------------------------------------------------

# Shared Components

## Navigation

-   AppSidebar
-   TopNavigation
-   Breadcrumbs
-   CommandPalette
-   SearchResults

## Data Display

-   BaseCard
-   PropertyGrid
-   DataTable -- the primary building block of a Collection View
    (docs/09-WORKSPACE-SPECIFICATIONS.md, "Collection View & Detail
    View"), paired with search/filter/sort controls
-   Timeline
-   StatusBadge
-   StatisticCard
-   RelationshipGraph
-   SectionCard (docs/08-DESIGN-SYSTEM.md) -- the standard building block
    for every Detail Workspace section

## Forms

-   FormLayout
-   PropertyEditor
-   AddressEditor
-   CustomerSelector
-   DeviceSelector
-   WorkflowSelector

## Actions

-   ActionMenu
-   ConfirmationDialog
-   BulkActionToolbar
-   WorkflowLauncher

------------------------------------------------------------------------

# Domain Components

Domain components should primarily configure shared components.

Examples:

``` text
CustomerCard
ServiceCard
DeviceCard
ONUCard
OLTCard
SplitterCard
WorkflowCard
PluginCard
```

Avoid embedding business logic inside these components.

------------------------------------------------------------------------

# Layout Components

Reusable layout primitives include:

``` text
SplitLayout
CardGrid
DetailsLayout
InspectorLayout
TimelineLayout
WizardLayout
EmptyStateLayout
```

`DetailsLayout` is the generic layout primitive underlying a Detail
Workspace; `DetailWorkspace` (see "Workspace Architecture") is the
specific workspace-level composition built from it -- header, Contents
navigation, and a single continuous column of collapsible sections, not
a page-plus-sidebar arrangement. A Collection View is typically assembled
from `CardGrid` or a `DataTable` (see "Shared Components," Data Display)
plus filter/sort controls, not `DetailsLayout`.

Pages should be assembled from these layouts instead of inventing new
structures.

------------------------------------------------------------------------

# State Ownership

  State               Owner
  ------------------- ------------------------
  Authentication      App
  Theme               App
  Navigation          App
  Current Workspace   Router
  Server Data         Query Layer
  Selected Entity     Workspace
  Section Collapse    User preference (persisted, per docs/02-DESIGN-PRINCIPLES.md principle 6)
  Dialog Visibility   Component
  Form State          Component / Composable

Avoid duplicating server state.

------------------------------------------------------------------------

# Plugin Extension Points

Plugins may contribute to:

-   Workspace Header
-   Sections (a plugin may contribute an additional Section to a Detail
    Workspace's SectionContainer)
-   Contents Navigation
-   Action Bar
-   Dashboard Widgets
-   Navigation
-   Search Providers
-   Property Editors
-   Diagnostics Panels

"Workspace Tabs" was previously listed here; Detail Workspaces do not
use tabs (docs/09-WORKSPACE-SPECIFICATIONS.md, "Tabs," under "Detail
Workspace Structure"), so contributing a Section is the extension point
a plugin uses instead. "Workspace Summary," "Relationship Panel," and
"Timeline" are consolidated into "Sections" for the same reason (see
"Workspace Architecture" above).

Plugins should register contributions through extension points rather
than modifying existing components.

------------------------------------------------------------------------

# Component Naming

## Base Components

Low-level UI primitives.

Examples:

-   BaseButton
-   BaseCard
-   BaseIcon
-   BaseInput
-   BaseModal

### BaseIcon and the Icon Library

Lucide is the official icon library for Palladium OSS
(docs/08-DESIGN-SYSTEM.md section 9), but no component may import it
directly except `BaseIcon` itself.

-   `BaseIcon` accepts a stable, application-defined icon name (plus
    size, color via CSS, and an optional stroke width) and maps that
    name to the matching Lucide icon internally.
-   Every other component -- base, app, navigation, workspace, domain,
    or plugin -- renders icons exclusively through `BaseIcon`.
-   `BaseIcon` never exposes a Lucide type, component, or prop shape to
    its callers.

This isolates the icon library behind one API: replacing Lucide with a
different library should only ever require changes inside `BaseIcon`.

## App Components

Application shell.

Examples:

-   AppSidebar
-   AppHeader
-   AppSearch

## Workspace Components

Shared workspace primitives.

Examples:

-   DetailWorkspace
-   WorkspaceHeader
-   ContentsNavigation
-   SectionContainer

## Domain Components

Examples:

-   CustomerCard
-   DeviceSummary
-   WorkflowStatus

## Plugin Components

Examples:

-   PluginPanel
-   PluginWidget
-   PluginAction

------------------------------------------------------------------------

# Recommended Folder Structure

``` text
src/
├── components/
│   ├── app/
│   ├── base/
│   ├── navigation/
│   ├── workspace/
│   ├── dashboard/
│   ├── data-display/
│   ├── forms/
│   ├── dialogs/
│   ├── customer/
│   ├── service/
│   ├── device/
│   ├── workflow/
│   └── plugin/
├── composables/
├── services/
├── stores/
├── router/
└── views/
```

------------------------------------------------------------------------

# Architectural Rules

1.  Every Entity Workspace's Detail Workspace uses the shared
    `DetailWorkspace` structure (header, Contents navigation, a single
    continuous column of sections -- see "Workspace Architecture").
    Landing Workspaces (Dashboard in Version 1) do not -- see "Workspace
    Archetypes."
2.  New reusable patterns should become shared components.
3.  Business logic must not live inside presentation components.
4.  Components should remain small and focused.
5.  Plugins must use extension points.
6.  Tables are for collections; cards are for individual objects --
    concretely, a Collection View is built from tables, a Detail View
    from cards (docs/09-WORKSPACE-SPECIFICATIONS.md, "Collection View &
    Detail View").
7.  Relationship navigation should be available wherever practical.
8.  Favor composition over duplication.
9.  Third-party icon libraries are isolated behind BaseIcon; no other
    component may import one directly.

------------------------------------------------------------------------

# Future Evolution

This document should evolve as Palladium OSS grows.

New domains should be implemented using existing primitives whenever
possible. New reusable UI patterns should be promoted into the shared
component library before being duplicated across multiple workspaces.

Maintaining a consistent operator experience is a primary architectural
goal.
