---
document: 03-DOMAIN-MODEL
status: Draft
title: Domain Model
version: 1.0-draft
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
6.  Asset
7.  Equipment Assignment
8.  Site
9.  Relationship Overview

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
-   Asset
-   Equipment Assignment
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
-   Contact information
-   Status
-   Billing reference (external)
-   Associated services

A Customer **owns Services**.

Customers do **not** directly own network equipment.

Equipment is associated through services.

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

# 6. Asset

An Asset represents a managed piece of physical network equipment that
Palladium is capable of identifying, tracking, and operating.

Version 1 recognizes three primary asset types:

-   OLT
-   ONU
-   Router

Future asset types may be added without changing the core domain model.

## Responsibilities

An Asset is responsible for maintaining:

-   Identity
-   Vendor
-   Model
-   Serial number
-   Operational status
-   Physical location
-   Current assignment

Assets exist independently of customers.

An ONU sitting on a shelf is still an asset.

A router awaiting deployment is still an asset.

------------------------------------------------------------------------

# 7. Equipment Assignment

Equipment Assignment represents the relationship between a Service and
one or more Assets.

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

Equipment Assignment records:

-   Assigned service
-   Assigned asset
-   Assignment date
-   Removal date
-   Assignment reason
-   Current status

Only one active assignment should exist for a given asset at a time.

Historical assignments are never deleted.

------------------------------------------------------------------------

# 8. Site

A Site represents a physical location operated by the ISP.

Examples include:

-   Central Office
-   POP
-   Headend
-   Remote Cabinet
-   Data Center

Sites provide operational context for assets.

A Site may contain:

-   OLTs
-   Routers
-   Network infrastructure
-   Supporting equipment

Customers are not located at Sites.

Customer service addresses are modeled separately as part of a Service.

------------------------------------------------------------------------

# 9. Relationship Overview

The core relationships within Palladium are intentionally
straightforward.

Customer → owns one or more Services

Service → has one or more Equipment Assignments

Equipment Assignment → references exactly one Asset

Asset → belongs to one Site

This structure separates business relationships from physical
infrastructure, allowing equipment to be reassigned without altering
customer or service history.

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
-   Asset assigned

Events provide the chronological history of Palladium.

They should never contain business logic.

Instead, they describe facts.

Future automation, reporting, timelines, and auditing all depend upon a
complete and trustworthy event history.

------------------------------------------------------------------------

# Architectural Principle

Workflows create Events.

Events do not execute Workflows.

This one-way relationship keeps execution separate from historical
record and avoids circular dependencies.

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
-   Assets
-   Equipment Assignments
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

Prospect → Active → Suspended → Closed

## Service

Requested → Provisioning → Active → Suspended → Decommissioned

## Asset

Inventory → Assigned → In Service → Maintenance → Retired

## Workflow Instance

Pending → Running → Completed \| Failed \| Cancelled

## Event

Created → Immutable Archive

Entity state changes should always be represented explicitly and should
generate corresponding operational events.

------------------------------------------------------------------------

# 17. Domain Invariants

The following rules must always remain true.

-   A Customer may own multiple Services.
-   A Service belongs to exactly one Customer.
-   An Asset may exist without being assigned.
-   An Asset may have only one active Equipment Assignment at a time.
-   Historical Equipment Assignments are never deleted.
-   Workflow definitions are immutable once published.
-   Workflow Instances are permanent historical records.
-   Events are immutable.
-   Vendor Plugins never contain business rules.

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

------------------------------------------------------------------------

# Related Documents

-   01-VISION.md
-   02-DESIGN-PRINCIPLES.md
-   04-NAVIGATION.md
-   05-WORKFLOW-ENGINE.md

------------------------------------------------------------------------

**End of Document**