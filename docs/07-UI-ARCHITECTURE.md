---
document: 07-UI-ARCHITECTURE
status: Draft
title: UI Architecture
version: 1.2-draft
---

# UI Architecture

## Executive Summary

The Palladium user interface is designed around operational workflows
rather than traditional CRUD pages.

Instead of forcing operators to navigate deep menu hierarchies, the
interface presents information through focused workspaces that bring
together everything needed to understand and complete a task.

This document defines the structural architecture of the frontend,
including workspaces, layouts, navigation patterns, and interaction
principles.

------------------------------------------------------------------------

# Table of Contents

1.  Purpose
2.  Design Goals
3.  Guiding Principles
4.  Core UI Concepts

------------------------------------------------------------------------

# 1. Purpose

The UI Architecture establishes a consistent structure for every screen
in Palladium.

It defines:

-   Layouts
-   Workspaces
-   Panels
-   Dialogs
-   Navigation
-   State management
-   Interaction patterns

The goal is to ensure every feature feels like part of a single cohesive
application.

------------------------------------------------------------------------

# 2. Design Goals

The interface should:

-   Prioritize operator efficiency.
-   Minimize context switching.
-   Surface relevant information immediately.
-   Keep navigation predictable.
-   Scale from small ISPs to large providers.
-   Support keyboard-first operation where practical.

The UI should reduce cognitive load rather than simply expose
functionality.

------------------------------------------------------------------------

# 3. Guiding Principles

The interface should be:

-   Search-first
-   Workspace-oriented
-   Task-focused
-   Consistent
-   Responsive
-   Observable

Every screen should answer three questions:

-   What am I looking at?
-   What can I do next?
-   What changed?

------------------------------------------------------------------------

# 4. Core UI Concepts

The frontend is built from a small set of reusable concepts:

-   Application Shell
-   Workspace
-   Panel
-   View
-   Dialog
-   Drawer
-   Timeline
-   Notification

Features should compose these building blocks rather than inventing new
interaction patterns.

------------------------------------------------------------------------

# Architect's Note

Operators should think in terms of customers, services, devices, and
workflows---not pages.

The interface should follow the operator's mental model instead of
requiring the operator to learn the application's internal organization.

# 5. Application Shell

The Application Shell provides the persistent framework for every screen
in Palladium.

It should remain visible as operators move between workspaces and
includes:

-   Global navigation
-   Search
-   Notifications
-   Active workflows
-   User menu
-   Status indicators

The shell should rarely reload, creating a fast and predictable
experience.

------------------------------------------------------------------------

# 6. Global Navigation

Navigation should expose major operational areas rather than individual
pages.

Examples include:

-   Dashboard
-   Customers
-   Services
-   Inventory
-   Monitoring
-   Workflows
-   Administration

Most day-to-day navigation should originate from global search rather
than menu traversal.

------------------------------------------------------------------------

# 7. Workspace Architecture

A Workspace is the primary unit of interaction.

Each workspace is centered around a single entity or operational task.

Examples:

-   Customer Workspace
-   Service Workspace
-   Device Workspace
-   OLT Workspace
-   Site Workspace
-   Workflow Workspace

A workspace aggregates everything relevant to its subject without
forcing operators to navigate elsewhere. Concretely, an Entity
Workspace's Detail Workspace is a single continuous, section-based page,
not tabs or nested pages (docs/02-DESIGN-PRINCIPLES.md, principle 6,
"Single-Workspace Operations"; docs/09-WORKSPACE-SPECIFICATIONS.md,
"Detail Workspace Structure").

------------------------------------------------------------------------

# 8. Workspace Lifecycle

Workspaces may be:

1.  Opened from search
2.  Opened from links
3.  Restored from history
4.  Reused if already open
5.  Closed by the operator

State should be preserved while a workspace remains open to reduce
repetitive navigation.

------------------------------------------------------------------------

# 9. Panel System

On a Detail Workspace, these are **sections** -- each built from
`SectionCard` (docs/08-DESIGN-SYSTEM.md), independently collapsible and
expanded by default (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail
Workspace Structure").

Typical sections include:

-   Summary
-   Timeline
-   Properties
-   Active Alarms
-   Related Resources
-   Running Workflows
-   Recent Activity

Sections should be independently refreshable and reusable across
multiple workspace types.

------------------------------------------------------------------------

# 10. Persistent Context

Operators should never lose context unnecessarily.

The UI should preserve:

-   Filters
-   Sort order
-   Scroll position
-   Expanded/collapsed sections
-   Workspace history

Detail Workspaces have no tabs to preserve selection for
(docs/09-WORKSPACE-SPECIFICATIONS.md, "Tabs," under "Detail Workspace
Structure") -- scroll position and each section's collapse state serve
the same "return to where I was" purpose instead.

Preserving context reduces cognitive load and improves efficiency.

------------------------------------------------------------------------

# 11. Multi-Workspace Behavior

Operators frequently investigate multiple related entities.

The interface should support multiple simultaneously open workspaces
with fast switching.

Opening a related entity should not force the current workspace to close
unless explicitly requested.

------------------------------------------------------------------------

# Design Principle

Navigation changes location.

Workspaces preserve context.

# 12. Views & Layout Regions

Each Workspace is composed of one or more Views arranged into
predictable layout regions.

A Detail Workspace's regions are a single continuous column, not a
page-plus-sidebar arrangement (docs/09-WORKSPACE-SPECIFICATIONS.md,
"Detail Workspace Structure"):

-   Header
-   Contents Navigation (in-page only, not a content sidebar)
-   Sections (Primary Content, Secondary Information, Timeline, and so
    on are all sections within the same column)
-   Footer Actions

Layouts should remain consistent across workspace types to reduce
cognitive load.

------------------------------------------------------------------------

# 13. Detail Workspace Interaction Model

Every Detail Workspace uses the same interaction model
(docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure";
docs/02-DESIGN-PRINCIPLES.md, principle 6, "Single-Workspace
Operations"):

-   Sticky Contents navigation -- remains visible while scrolling when
    practical.
-   Active section highlighting -- Contents navigation highlights
    whichever section is currently in view.
-   Smooth scrolling -- selecting an item in Contents navigation scrolls
    to that section rather than jumping or navigating away.
-   Collapsible sections -- an operator may collapse a section
    individually; collapse state may be remembered as a user preference.
-   All sections expanded by default -- an operator should see the whole
    object without acting first.

This pattern is reused consistently across the Customer, Device,
Service, Inventory, Network, and future Detail Workspaces. It is not
reinvented per workspace.

------------------------------------------------------------------------

# 14. Dialogs & Drawers

Dialogs are used for focused, interruptive tasks requiring explicit
confirmation.

Examples include:

-   Delete confirmation
-   Password reset
-   Workflow approval
-   Service activation

Drawers provide supplemental information or editing without leaving the
current workspace.

Examples include:

-   Entity details
-   Related resources
-   Event inspection
-   Configuration preview

Use dialogs for decisions and drawers for exploration.

------------------------------------------------------------------------

# 15. Tables & Data Presentation

Tables should prioritize scanning over density.

A table is the primary mechanism of a Collection View
(docs/09-WORKSPACE-SPECIFICATIONS.md, "Collection View & Detail View"):
it should display only identifying information, not the full object.

Common capabilities include:

-   Sorting
-   Filtering
-   Column selection
-   Saved views
-   Bulk actions
-   Keyboard navigation

Every table should support opening the selected item directly into its
Detail View.

------------------------------------------------------------------------

# 16. Forms & Editing Patterns

Forms should be task-oriented rather than database-oriented.

Guidelines:

-   Group related fields together.
-   Validate as early as practical.
-   Clearly distinguish required and optional fields.
-   Explain validation failures in plain language.
-   Preview impactful changes before submission when appropriate.

Long-running operations should execute through the Workflow Engine
rather than directly through form submission.

------------------------------------------------------------------------

# 17. Workflow Integration

Workflows are first-class UI elements.

Operators should be able to:

-   Launch workflows
-   Monitor progress
-   View logs
-   Inspect step results
-   Retry failed workflows
-   Navigate to affected resources

Workflow progress should remain visible regardless of which workspace is
currently active.

------------------------------------------------------------------------

# 18. Notifications & Activity

Notifications communicate changes requiring operator awareness.

Categories include:

-   Information
-   Success
-   Warning
-   Error

Every notification should link to the relevant Workspace, Workflow, or
Event when applicable.

------------------------------------------------------------------------

# 19. State Management

Application state should be separated into distinct layers:

-   Global application state
-   Workspace state
-   View state
-   Temporary UI state

This separation reduces unnecessary re-rendering and keeps state
predictable.

------------------------------------------------------------------------

# Design Principle

Every interaction should either:

-   Move the operator forward, or
-   Improve the operator's understanding.

The interface should never create unnecessary work.

# 20. Loading & Empty States

Loading states should communicate progress without disrupting the
operator's workflow.

Guidelines:

-   Use skeleton placeholders for content-heavy views.
-   Show progress indicators for long-running operations.
-   Preserve existing content while refreshing whenever practical.

Empty states should explain why no data is available and provide a clear
next action.

------------------------------------------------------------------------

# 21. Error & Recovery UX

Errors should help operators recover, not merely report failure.

Every error should:

-   Explain what happened
-   Describe the impact
-   Suggest corrective actions
-   Link to related events or workflows when available

Technical details may be available on demand but should not overwhelm
routine operations.

------------------------------------------------------------------------

# 22. Accessibility

Accessibility is a core quality attribute.

The interface should:

-   Support keyboard-only navigation
-   Provide meaningful labels
-   Maintain sufficient color contrast
-   Work with screen readers
-   Avoid color as the sole indicator of status

Accessibility improvements should benefit all operators, not only those
using assistive technologies.

------------------------------------------------------------------------

# 23. Keyboard Navigation

Common operations should be available without a mouse.

Examples include:

-   Global search
-   Workspace switching
-   Table navigation
-   Workflow launch
-   Command shortcuts

Keyboard shortcuts should remain consistent throughout the application.

------------------------------------------------------------------------

# 24. Responsive Behavior

Palladium is designed primarily for desktop operations but should remain
usable on smaller displays.

The layout should adapt by collapsing secondary panels, drawers, and
navigation while preserving access to all core functionality.

------------------------------------------------------------------------

# 25. Performance Considerations

The UI should remain responsive even when managing large networks.

Strategies include:

-   Incremental data loading
-   Virtualized tables
-   Background refresh
-   Optimistic UI where appropriate
-   Intelligent caching

Performance should be treated as a feature, not an optimization added
later.

------------------------------------------------------------------------

# 26. Future Enhancements

Future versions may include:

-   Command palette
-   Custom dashboards
-   User-defined workspace layouts
-   Offline-friendly capabilities
-   Real-time collaborative views
-   AI-assisted operational guidance

These enhancements should build upon the same architectural principles
defined in this document.

------------------------------------------------------------------------

# Closing Statement

The UI Architecture defines how operators experience Palladium.

By organizing the application around workspaces, consistent interaction
patterns, and operational workflows, the interface becomes an extension
of the operator's mental model rather than an obstacle to it.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft
  1.1 Draft   2026-07-30   Clarified tables belong to Collection Views and open a Detail View (section 14)
  1.2 Draft   2026-07-30   Added Detail Workspace Interaction Model (section 13): sticky Contents navigation, active section highlighting, smooth scrolling, collapsible sections; replaced tab/sidebar language with sections throughout

------------------------------------------------------------------------

# Related Documents

-   02-DESIGN-PRINCIPLES.md
-   04-NAVIGATION.md
-   05-WORKFLOW-ENGINE.md
-   06-PLUGIN-ARCHITECTURE.md
-   08-DESIGN-SYSTEM.md
-   09-WORKSPACE-SPECIFICATIONS.md

**End of Document**