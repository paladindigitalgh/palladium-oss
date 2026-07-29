---
document: 02-DESIGN-PRINCIPLES
status: Draft
title: Design Principles
version: 1.0-draft
---

# Design Principles

## Executive Summary

This document defines the enduring principles that guide the design,
architecture, and evolution of Palladium. These principles are intended
to outlive individual features and implementation details. Every
architectural decision, workflow, user interface, and subsystem should
be traceable back to one or more of these principles.

These principles form the constitutional foundation of the project.

------------------------------------------------------------------------

# Table of Contents

1.  Purpose
2.  Operations First
3.  Search First
4.  Workspaces, Not Pages
5.  Workflows Over Buttons
6.  Business Intent Over Vendor Commands
7.  Read Before Write
8.  Events as Operational History
9.  Correlation Over Collection
10. Simplicity Over Feature Count
11. Consistency Above Cleverness

------------------------------------------------------------------------

# 1. Purpose

Design principles exist to answer a simple question:

> "How should we decide?"

When multiple implementations are technically correct, these principles
determine which approach best supports the long-term vision of
Palladium.

They are intentionally stable. Features may evolve, technologies may
change, but these principles should rarely require modification.

------------------------------------------------------------------------

# 2. Operations First

Palladium exists to support operational work.

Every screen, action, and workflow should solve a real problem
encountered by network operators.

The platform should never prioritize technical elegance over operational
usefulness.

Questions to ask:

-   Does this reduce operator effort?
-   Does this improve operational confidence?
-   Does this make a common task easier?

If the answer is no, the feature should be reconsidered.

------------------------------------------------------------------------

# 3. Search First

Operators begin with information they already know:

-   Customer name
-   Service address
-   ONU serial number
-   Router hostname
-   OLT name

Search should immediately lead them into the appropriate operational
workspace.

Navigation exists for exploration.

Search exists for work.

------------------------------------------------------------------------

# Architect's Note

The best interface is often a search bar.

# 4. Workspaces, Not Pages

Traditional applications organize functionality into pages. Operators
are expected to navigate between those pages to gather information and
perform actions.

Palladium adopts a different philosophy.

Every major interface is a **workspace**.

A workspace is centered around an operational subject, such as a
customer, service, OLT, or asset. It brings together everything an
operator needs to understand the current state, review history, identify
relationships, and safely take action.

A workspace should include:

-   Current state
-   Historical activity
-   Related objects
-   Available workflows
-   Recent events
-   Operational notes

The objective is simple: complete the task without unnecessary
navigation.

------------------------------------------------------------------------

# 5. Workflows Over Buttons

Individual buttons encourage isolated actions.

Operational workflows encourage deliberate execution.

Rather than exposing low-level commands, Palladium guides operators
through validated, repeatable processes.

Every workflow should provide:

-   Validation before execution
-   Visibility into progress
-   Clear success and failure states
-   Automatic audit logging
-   Meaningful rollback guidance where applicable

Complexity belongs inside the workflow---not in the operator's head.

------------------------------------------------------------------------

# 6. Business Intent Over Vendor Commands

Operators think in business outcomes.

They do not think in CLI syntax or vendor APIs.

Examples include:

-   Replace ONU
-   Upgrade Service
-   Move Customer
-   Reboot Equipment
-   Run Diagnostics

Palladium translates these intentions into vendor-specific
implementations through plugins.

This abstraction allows the platform to support multiple vendors while
presenting a consistent operational experience.

Vendor complexity should remain behind the interface.

------------------------------------------------------------------------

# Architectural Implications

This principle requires a clear separation between:

-   Domain logic
-   Workflow engine
-   Vendor plugins
-   Transport mechanisms

The core platform should never depend on vendor-specific command syntax.

# 7. Read Before Write

Palladium should prefer understanding the current state of the network
before attempting to change it.

Whenever practical, workflows should begin by reading and validating the
existing configuration before issuing any write operations.

This principle serves several purposes:

-   Confirms the current operational state.
-   Detects configuration drift.
-   Prevents unnecessary changes.
-   Provides meaningful information to the operator before execution.
-   Creates opportunities to identify problems before they become
    failures.

For example, a service upgrade workflow should first retrieve the
customer's current provisioning profile, assigned equipment, service
state, and any active alarms before calculating the required changes.

Understanding precedes action.

------------------------------------------------------------------------

# 8. Events as Operational History

Every meaningful action within Palladium should generate an event.

Events form the operational history of the platform.

Examples include:

-   Customer created
-   Service provisioned
-   ONU replaced
-   Workflow completed
-   Workflow failed
-   Diagnostics executed
-   Equipment reassigned
-   Configuration synchronized

Rather than simply recording that something exists, Palladium records
what happened.

This provides a chronological history of network operations that
supports troubleshooting, auditing, and future analytics.

Events should be immutable.

Corrections should generate new events rather than modifying historical
records.

------------------------------------------------------------------------

# Architectural Implications

Events become the foundation for:

-   Activity timelines
-   Audit history
-   Notifications
-   Future automation
-   Reporting
-   Operational analytics

The event stream should describe the operational life of the network.

------------------------------------------------------------------------

# 9. Correlation Over Collection

Collecting more information does not necessarily improve operations.

Operators need context---not volume.

Palladium should prioritize connecting related information rather than
displaying every available metric.

Examples include:

-   Displaying the customer alongside their services, assigned
    equipment, recent events, and active workflows.
-   Showing ONU optical levels together with provisioning status and
    recent alarms.
-   Presenting diagnostic results in the context of the affected service
    rather than as isolated output.

The goal is to answer questions, not simply expose data.

Every piece of information shown within a workspace should contribute to
operator understanding.

------------------------------------------------------------------------

# Architect's Note

Information without context increases cognitive load.

Correlated information reduces it.

# 10. Simplicity Over Feature Count

Palladium should not compete by offering the largest number of features.

Instead, it should compete by making the most important operational
tasks intuitive, reliable, and predictable.

Every proposed feature should answer three questions:

1.  Does it solve a real operational problem?
2.  Can it be understood by a new operator?
3.  Does it fit naturally within the existing workflows?

If the answer to any of these questions is no, the feature should be
reconsidered.

Complexity should be earned, not accumulated.

------------------------------------------------------------------------

# 11. Consistency Above Cleverness

Operators benefit from consistency more than novelty.

The same concepts should behave the same way throughout the platform.

Examples include:

-   Every workspace presents state, history, relationships, and
    workflows.
-   Every workflow follows the same lifecycle.
-   Every search result opens into an appropriate workspace.
-   Every event appears in a consistent timeline.

Predictability reduces training time, minimizes errors, and builds
confidence.

------------------------------------------------------------------------

# 12. Design Decision Checklist

Before implementing a new feature, ask:

-   Does it support a real operational workflow?
-   Does it reduce context switching?
-   Does it increase operational confidence?
-   Does it fit naturally into an existing workspace?
-   Does it follow business intent rather than vendor implementation?
-   Does it preserve simplicity?
-   Will operators immediately understand it?
-   Does it generate meaningful operational events?

If several answers are "no," the design should be revised before
implementation.

------------------------------------------------------------------------

# 13. Applying These Principles

These principles are intended to guide every stage of development.

They should influence:

-   Product planning
-   User interface design
-   Domain modeling
-   Workflow implementation
-   Plugin architecture
-   Code reviews
-   Feature prioritization

When uncertainty exists, these principles take precedence over
convenience.

The preferred solution is the one that best reinforces the long-term
vision of Palladium.

------------------------------------------------------------------------

# Closing Statement

Design principles are valuable only when they influence decisions.

Every feature, workflow, and architectural change should be evaluated
against this document.

If a proposal conflicts with these principles, the proposal---not the
principles---should be questioned first.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft

------------------------------------------------------------------------

# Related Documents

-   00-INTRODUCTION.md
-   01-VISION.md
-   03-DOMAIN-MODEL.md

------------------------------------------------------------------------

**End of Document**
