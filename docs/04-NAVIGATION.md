---
document: 04-NAVIGATION
status: Draft
title: Navigation
version: 1.0-draft
---

# Navigation

## Executive Summary

Navigation is more than a menu system---it is the framework through
which operators understand and interact with the entire platform.

Palladium is designed around operational workspaces rather than isolated
pages. Operators should spend their time solving network problems, not
searching for functionality.

This document defines the information architecture, navigation
philosophy, and interaction model that guide every screen within
Palladium.

------------------------------------------------------------------------

# Table of Contents

1.  Navigation Philosophy
2.  Design Goals
3.  Information Architecture
4.  Global Navigation
5.  Search
6.  Workspace Navigation
7.  URL Strategy
8.  Future Considerations

------------------------------------------------------------------------

# 1. Navigation Philosophy

Navigation should disappear into the background.

An experienced operator should rarely think about where information is
located. Instead, they should move naturally between operational tasks.

Three ideas drive every navigation decision:

-   Search is the primary entry point.
-   Workspaces are the primary destination.
-   Context should never be unnecessarily lost.

------------------------------------------------------------------------

# 2. Design Goals

The navigation system should:

-   Minimize clicks.
-   Reduce context switching.
-   Keep related information together.
-   Make common tasks immediately accessible.
-   Remain predictable throughout the application.
-   Scale as Palladium grows without becoming cluttered.

Whenever there is a tradeoff between exposing more features and
maintaining clarity, clarity should win.

------------------------------------------------------------------------

# 3. Information Architecture

Palladium organizes information into three layers.

## Global Layer

Information and actions available everywhere.

Examples include:

-   Global Search
-   Notifications
-   User Profile
-   Settings

## Workspace Layer

The primary working area for a specific operational subject.

Examples include:

-   Customer Workspace
-   Service Workspace
-   OLT Workspace
-   Asset Workspace

## Context Layer

Supporting information related to the active workspace.

Examples include:

-   Activity timeline
-   Related objects
-   Recent workflows
-   Notes
-   Diagnostics

------------------------------------------------------------------------

# Architect's Note

Operators think in tasks, not menu hierarchies.

Navigation should reflect the way operators work rather than the way
software is organized.

# 4. Global Navigation

Global navigation should remain intentionally small and stable.

Operators should not navigate through long menus to locate operational
tools.

The primary navigation consists of:

-   Dashboard
-   Customers
-   Services
-   Devices
-   Network
-   Inventory
-   Explorer
-   Administration

Each destination represents a major operational workspace rather than a
collection of unrelated pages.

## Persistent Navigation

Global navigation should remain available regardless of the current
workspace.

Operators should always be able to:

-   Search
-   Return to the dashboard
-   Switch workspaces
-   Review notifications
-   Access their profile

without losing their current context.

------------------------------------------------------------------------

# 5. Search

Search is the primary navigation mechanism within Palladium.

Operators should never need to know where information is stored before
finding it.

Search should support:

-   Customer names
-   Service IDs
-   ONU serial numbers
-   Router hostnames
-   OLT names
-   Asset tags
-   Addresses
-   Workflow identifiers

Results should be grouped by entity type and display enough context to
identify the correct object immediately.

Selecting a result should always open the appropriate workspace.

## Search Principles

-   Search before browse.
-   Fuzzy matching where appropriate.
-   Keyboard-first interaction.
-   Fast enough to feel instantaneous.
-   Preserve recent searches for convenience.

------------------------------------------------------------------------

# 6. Workspace Navigation

Every workspace follows the same high-level layout.

A typical workspace contains:

-   Summary
-   Activity Timeline
-   Relationships
-   Available Workflows
-   Diagnostics
-   Notes
-   Configuration (where applicable)

Operators should not need to learn a different navigation pattern for
each workspace.

Consistency reduces cognitive load and training time.

------------------------------------------------------------------------

# Design Decision

Opening a workspace should answer the question:

"What does the operator most likely need next?"

The layout should prioritize those answers before secondary information.

# 7. URL & Routing Strategy

URLs should identify operational resources rather than user interface
layouts.

Examples:

-   `/customers/{id}`
-   `/services/{id}`
-   `/assets/{id}`
-   `/olts/{id}`
-   `/workflows/{id}`

A bookmarked URL should always restore the same workspace and context
whenever practical.

Nested resources should preserve their relationship while remaining
directly addressable.

Examples:

-   `/customers/{id}/services`
-   `/olts/{id}/onus`

Human-readable URLs are preferred where they do not compromise
uniqueness.

------------------------------------------------------------------------

# 8. Context Preservation

Operators frequently investigate multiple issues simultaneously.

Navigation should preserve context whenever possible.

Examples include:

-   Remembering scroll position.
-   Preserving search filters.
-   Restoring open tabs.
-   Returning to the previous workspace state.
-   Maintaining selected timeline ranges.

Changing workspaces should not force operators to repeat work
unnecessarily.

------------------------------------------------------------------------

# 9. Permission-Aware Navigation

Navigation should reflect what an operator is authorized to do.

Users should not see workflows or administrative functions they cannot
access.

Permissions should affect:

-   Visible navigation items
-   Workspace actions
-   Administrative settings
-   Search results (where appropriate)

Permission enforcement must always occur on the server regardless of
client-side visibility.

------------------------------------------------------------------------

# 10. Keyboard Shortcuts

Common operational tasks should be accessible without leaving the
keyboard.

Recommended shortcuts include:

-   `/` --- Focus global search
-   `Esc` --- Close dialogs
-   `Ctrl+K` --- Command palette
-   `[` and `]` --- Previous/next workspace history
-   `?` --- Show shortcut reference

Shortcuts should accelerate experienced operators without becoming
mandatory.

------------------------------------------------------------------------

# 11. Responsive Behavior

Palladium is primarily designed for desktop use in network operations
centers.

Tablet support should preserve core workflows.

Mobile support should prioritize read-only access and lightweight
operational tasks until dedicated technician workflows are introduced.

Information hierarchy must remain consistent regardless of screen size.

------------------------------------------------------------------------

# Design Principle

Navigation should adapt to the operator.

Operators should never adapt to the navigation.

# 12. Future Navigation Enhancements

The initial navigation model is intentionally conservative.

As Palladium evolves, navigation may expand to support additional
capabilities while preserving the core principles established in this
document.

Potential future enhancements include:

-   Command Palette with operational actions
-   Recently Viewed workspaces
-   Favorite Customers, Services, and Assets
-   Pinned Workspaces
-   Workspace templates
-   Multi-workspace split view
-   Global activity feed
-   Cross-workspace breadcrumbs
-   User-customizable dashboards
-   AI-assisted navigation and recommendations

These enhancements should improve operator efficiency without increasing
cognitive complexity.

------------------------------------------------------------------------

# 13. Navigation Decision Checklist

Before introducing a new navigation element, ask:

-   Does it reduce operator effort?
-   Does it reduce context switching?
-   Does it help complete an operational task?
-   Is it consistent with existing navigation?
-   Can an experienced operator predict where it belongs?
-   Does it avoid unnecessary clicks?
-   Can it be discovered through search instead?
-   Will it still make sense as Palladium grows?

If the answer to several of these questions is "no," the navigation
should be redesigned before implementation.

------------------------------------------------------------------------

# Closing Statement

Navigation is not simply a way to move through Palladium.

It is a reflection of the product's operational philosophy.

Every menu, search result, workspace, and interaction should reinforce a
single idea:

Operators should focus on solving network problems---not locating
software features.

When navigation becomes invisible, operators become more effective.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft

------------------------------------------------------------------------

# Related Documents

-   01-VISION.md
-   02-DESIGN-PRINCIPLES.md
-   03-DOMAIN-MODEL.md
-   05-WORKFLOW-ENGINE.md

------------------------------------------------------------------------

**End of Document**
