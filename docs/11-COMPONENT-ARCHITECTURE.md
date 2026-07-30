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
Plugin, Site, etc.) should feel structurally identical.

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

# Workspace Architecture

Every domain workspace should follow the same layout.

``` text
Workspace
├── WorkspaceHeader
├── WorkspaceActions
├── WorkspaceSummary
├── WorkspaceContent
├── RelationshipPanel
├── TimelinePanel
└── InspectorPanel (optional)
```

## Workspace Header

Contains:

-   Object name
-   Status
-   Health indicators
-   Breadcrumbs
-   Primary actions

## Workspace Summary

Displays high-value information first.

Examples:

-   Customer contact details
-   ONU serial number
-   OLT model
-   Service package

## Workspace Content

The primary domain-specific information.

## Relationship Panel

Displays related entities.

Example:

Customer

↓

Service

↓

ONU

↓

PON

↓

OLT

↓

Cabinet

↓

Site

Relationships should always be navigable.

## Timeline Panel

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
-   DataTable
-   Timeline
-   StatusBadge
-   StatisticCard
-   RelationshipGraph

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
  Dialog Visibility   Component
  Form State          Component / Composable

Avoid duplicating server state.

------------------------------------------------------------------------

# Plugin Extension Points

Plugins may contribute to:

-   Workspace Header
-   Workspace Summary
-   Workspace Tabs
-   Relationship Panel
-   Timeline
-   Action Bar
-   Dashboard Widgets
-   Navigation
-   Search Providers
-   Property Editors
-   Diagnostics Panels

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

-   WorkspaceHeader
-   WorkspaceSummary
-   WorkspaceTimeline

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

1.  Every workspace uses the shared Workspace layout.
2.  New reusable patterns should become shared components.
3.  Business logic must not live inside presentation components.
4.  Components should remain small and focused.
5.  Plugins must use extension points.
6.  Tables are for collections; cards are for individual objects.
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
