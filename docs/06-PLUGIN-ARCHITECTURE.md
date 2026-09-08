---
document: 06-PLUGIN-ARCHITECTURE
status: Draft
title: Plugin Architecture
version: 1.2-draft
---

# Plugin Architecture

## Executive Summary

Palladium is designed to manage heterogeneous networks without embedding
vendor-specific logic into the core platform.

The Plugin Architecture provides a stable contract between the Palladium
core and vendor implementations. New vendors, protocols, and device
families can be added by implementing the plugin interfaces rather than
modifying the application itself.

This document defines the goals, boundaries, and core concepts of the
plugin system.

------------------------------------------------------------------------

## Implementation Status (v1)

Everything below this line describes the target design. What exists
today (see `internal/plugin/`) is a deliberately minimal slice of it:

-   A `Plugin` (`internal/plugin/plugin.go`) is just
    `Name()/Vendor()/Capabilities()/Execute()`. There is no
    manifest-based discovery, no Driver interface, no Transport
    abstraction, no Plugin Context object, and no version/compatibility
    negotiation.
-   The `Registry` is an in-memory map. Every plugin registers once at
    startup, before the HTTP server starts serving requests -- nothing
    is discovered or loaded at runtime, and there is no hot-swap or
    marketplace concept.
-   Two plugins are registered today: `internal/plugin/mock`, a
    simulated vendor, and `internal/provisioning/kontron/plugin`, a real
    (non-simulated) Kontron/Iskratel C16 integration -- Palladium's
    first real vendor Plugin. It declares
    `ProvisionService`/`ResumeService`/`SuspendService`/`DisconnectService`
    among `Capabilities()` and is registered after `mock` in
    `cmd/server/main.go`, so per `Registry.Register`'s
    last-write-wins-per-capability rule it now handles those four
    capabilities for real over SSH; `mock` still handles
    `ReprovisionService`/`SynchronizeService` (neither has real Kontron
    support yet) and remains the only plugin for every other vendor. No
    Nokia, Calix, Adtran, or MikroTik integration exists yet in any form.

Real Kontron config-change work goes well beyond what the Plugin wraps,
though. `internal/provisioning/kontron/plugin.Plugin` is a thin adapter
around `internal/provisioning/kontron/service.ServiceProfileService`
(`Apply`/`Remove`, applying or removing a Product's Kontron
service-profile on an ONU) -- everything else real and Kontron-specific
still bypasses `internal/plugin` entirely, as standalone REST actions
guarded by their own RBAC capabilities rather than Capability Model
dispatch:

-   `internal/diagnostics/kontron` is the read-only half: an
    interactive-shell SSH client (`internal/platform/ssh`, added for
    Kontron's exec-channel-less CLI and its "-- More --" pager) wrapped
    by nine vendor-specific `show` commands (including a network-wide,
    concurrent ONU-blacklist scan across every Kontron OLT, aggregated
    by `internal/diagnostics/kontron/service.KontronService`), exposed
    over its own HTTP routes (`internal/diagnostics/kontron/httpapi`)
    rather than through the Capability Model in section 7 below.
-   `internal/provisioning/kontron/service.AuthorizationService`
    ("Discover ONU" -- authorizing a physically-detected ONU:
    `configure` / `interface` / `onu serial-number` / `service-profile`
    / `exit` / `exit` / `save config`) and
    `DeauthorizationService` ("Deauthorize ONU" -- the mirror image,
    removing an ONU's base authorization and retiring its Device) are
    each triggered from a DeviceID a caller already has in hand, not a
    Service lifecycle transition, so neither goes through the Plugin
    interface either -- both guarded by `authz.CanRunProvisioning`.
-   `ServiceProfileService.Remove` is also called directly (not through
    the Plugin/workflow path) by `internal/customer/removal.RemovalService`
    ("Remove Customer"), to tear down real OLT state as part of that
    cascade -- a caller that already has the Service/Equipment in hand
    from a non-workflow context.

These were built this way because each need was narrow and specific
(ONU status for the Customer Workspace, bringing a newly-detected ONU
into service, tearing down one ONU's authorization from a Device Detail
page, cascading a Customer removal) rather than a generic Service
lifecycle transition needing multi-vendor Capability dispatch. Treat the
rest of this document as where the plugin system is headed for the
*Service lifecycle* capabilities the real Kontron Plugin now covers --
Provision/Resume/Suspend/Disconnect -- not as a description of every
Kontron-specific action in this codebase, several of which are
deliberately standalone REST by design and are expected to stay that
way (see internal/provisioning/kontron/service.DeauthorizationService
and AuthorizationService's own doc comments).

------------------------------------------------------------------------

# Table of Contents

1.  Purpose
2.  Design Goals
3.  Architectural Principles
4.  Core Concepts
5.  Plugin Lifecycle
6.  Registration & Discovery
7.  Capability Model
8.  Transport Abstraction
9.  Driver Interfaces
10. Resource Model
11. Operation Contracts
12. Plugin Context
13. Configuration & Secrets
14. Error Handling
15. Result Objects
16. Plugin Versioning
17. Compatibility Management
18. Testing & Simulation
19. Security Boundaries
20. Packaging & Deployment
21. Future Enhancements

------------------------------------------------------------------------

# 1. Purpose

The Plugin Architecture isolates vendor-specific behavior from the
Palladium core.

Examples include:

-   OLT management
-   ONU provisioning
-   Router configuration
-   ACS integration
-   Monitoring integrations
-   Inventory synchronization

The core platform understands business operations, while plugins
understand how to perform those operations for a specific technology or
vendor.

------------------------------------------------------------------------

# 2. Design Goals

The architecture should:

-   Keep the core vendor-neutral.
-   Support multiple vendors simultaneously.
-   Allow plugins to evolve independently.
-   Minimize breaking changes.
-   Expose a consistent operational interface.
-   Enable testing without physical hardware.

No plugin should require changes to the Palladium core simply to support
a new device family.

------------------------------------------------------------------------

# 3. Architectural Principles

Plugins should be:

-   Discoverable
-   Versioned
-   Self-describing
-   Capability-driven
-   Sandboxed from one another
-   Independently testable

The core should depend only on interfaces, never concrete vendor
implementations.

------------------------------------------------------------------------

# 4. Core Concepts

The Plugin Architecture is built around several concepts:

-   Plugin
-   Capability
-   Driver
-   Transport
-   Resource
-   Operation
-   Result

These concepts provide a common language for every integration
regardless of protocol or manufacturer.

------------------------------------------------------------------------

# Architect's Note

Palladium should never need to ask, "Is this a Kontron, Nokia, Adtran,
or MikroTik?"

Instead, it asks, "Which plugin provides the capability I need?"

# 5. Plugin Lifecycle

Every plugin follows a predictable lifecycle managed by the Palladium
core.

The lifecycle consists of:

1.  Discovery
2.  Registration
3.  Initialization
4.  Health Verification
5.  Capability Publication
6.  Active Operation
7.  Graceful Shutdown

Plugins should never manage their own lifecycle independently. The core
platform remains responsible for orchestration.

------------------------------------------------------------------------

# 6. Registration & Discovery

Plugins register themselves using a manifest that describes their
identity and capabilities.

The manifest should include:

-   Plugin name
-   Vendor
-   Version
-   Supported device families
-   Supported transports
-   Required configuration
-   Exposed capabilities

At startup, Palladium discovers available plugins, validates
compatibility, and registers them with the capability registry.

------------------------------------------------------------------------

# 7. Capability Model

Palladium does not invoke vendor-specific functions directly.

Instead, plugins advertise capabilities such as:

-   Provision Service
-   Replace ONU
-   Reboot Device
-   Read Inventory
-   Upgrade Firmware
-   Run Diagnostics
-   Retrieve Statistics

The workflow engine requests a capability, and the platform selects an
appropriate plugin to fulfill it.

A single capability may be implemented by many different plugins.

------------------------------------------------------------------------

# 8. Transport Abstraction

Communication protocols are implementation details.

Examples include:

-   SSH
-   SNMP
-   REST APIs
-   TR-069
-   NETCONF
-   gRPC
-   Serial Console

The core platform requests an operation, while the plugin chooses the
appropriate transport and protocol.

This abstraction allows workflows to remain technology-agnostic.

------------------------------------------------------------------------

# Design Principle

Business workflows should depend on capabilities---not vendors,
protocols, or device models.

# 9. Driver Interfaces

Plugins expose functionality through well-defined driver interfaces.

A driver is responsible for translating Palladium operations into
vendor-specific actions.

Examples include:

-   OLT Driver
-   ONU Driver
-   Router Driver
-   ACS Driver
-   Monitoring Driver

Drivers should implement common interfaces while hiding vendor-specific
implementation details.

The Palladium core interacts only with these interfaces, never directly
with vendor code.

------------------------------------------------------------------------

# 10. Resource Model

Plugins operate on Resources rather than raw identifiers.

Examples of resources include:

-   Customer
-   Service
-   Site
-   OLT
-   PON Port
-   ONU
-   Router
-   Interface

Resources provide the context necessary for a plugin to perform an
operation without exposing internal database structures.

------------------------------------------------------------------------

# 11. Operation Contracts

Every plugin operation follows a consistent contract.

An operation receives:

-   Context
-   Target resource
-   Requested capability
-   Input parameters

An operation returns:

-   Status
-   Result data
-   Generated events
-   Warnings
-   Errors

This consistent contract allows workflows to execute any plugin
operation without special-case logic.

------------------------------------------------------------------------

# 12. Plugin Context

Each operation executes within a Plugin Context supplied by the core
platform.

The context may include:

-   Authenticated operator
-   Workflow instance
-   Correlation ID
-   Logger
-   Configuration
-   Secrets
-   Cancellation token
-   Timeout information

Providing context centrally keeps plugins focused on business logic
instead of platform concerns.

------------------------------------------------------------------------

# 13. Configuration & Secrets

Plugins should never hardcode credentials or environment-specific
values.

Configuration should be declarative and validated during initialization.

Sensitive values such as passwords, API tokens, and private keys must be
supplied through Palladium's secret management facilities and never
written to logs.

------------------------------------------------------------------------

# 14. Error Handling

Plugins should return structured errors instead of vendor-specific
messages whenever possible.

Errors should distinguish between:

-   Validation failures
-   Connectivity failures
-   Authentication failures
-   Unsupported capabilities
-   Temporary failures
-   Permanent failures

This allows the Workflow Engine to make consistent retry and recovery
decisions.

------------------------------------------------------------------------

# 15. Result Objects

Successful operations return structured result objects.

Results may include:

-   Updated resource state
-   Metrics
-   Inventory changes
-   Generated events
-   Informational messages

Result objects should be predictable and stable across plugins
implementing the same capability.

------------------------------------------------------------------------

# Architectural Principle

Interfaces are the contract.

Plugins may differ internally, but they must appear identical to the
Palladium core.

# 16. Plugin Versioning

Plugins are independently versioned components.

Each plugin should publish:

-   Plugin version
-   Supported Palladium API version
-   Supported device families
-   Capability versions

Breaking changes must require a new major version.

The Palladium core should validate compatibility during startup and
refuse to load incompatible plugins.

------------------------------------------------------------------------

# 17. Compatibility Management

The plugin system should allow multiple vendors---and multiple
generations of the same vendor---to coexist.

Compatibility should be determined through declared capabilities and
supported interface versions rather than vendor-specific logic.

Deprecated capabilities may remain available for legacy devices while
newer plugins implement expanded functionality.

------------------------------------------------------------------------

# 18. Testing & Simulation

Every plugin should be testable without production hardware.

Recommended testing layers include:

-   Unit tests
-   Mock transports
-   Simulated devices
-   Integration tests
-   Vendor acceptance tests

Simulation enables rapid development, repeatable testing, and safer
releases.

------------------------------------------------------------------------

# 19. Security Boundaries

Plugins execute with the minimum privileges required.

Plugins should:

-   Access only required secrets
-   Validate all external input
-   Avoid persistent local state where possible
-   Never expose sensitive data through logs or errors

The core platform remains responsible for authentication, authorization,
and secret management.

------------------------------------------------------------------------

# 20. Packaging & Deployment

Plugins should be independently packaged and deployable.

Administrators should be able to:

-   Install new plugins
-   Upgrade existing plugins
-   Disable unused plugins
-   Inspect plugin health and version information

Plugin deployment should never require recompiling the Palladium core.

------------------------------------------------------------------------

# 21. Future Enhancements

Future versions may support:

-   Marketplace distribution
-   Hot-swappable plugins
-   Remote plugin repositories
-   Signed plugin packages
-   Capability negotiation
-   Automatic compatibility testing

These additions should extend---not fundamentally alter---the plugin
architecture.

------------------------------------------------------------------------

# Closing Statement

The Plugin Architecture allows Palladium to remain vendor-neutral while
supporting an expanding ecosystem of technologies.

By defining stable interfaces and capability-driven contracts, the
platform can evolve without coupling business workflows to individual
manufacturers or protocols.

------------------------------------------------------------------------

# Revision History

  Version     Date         Description
  ----------- ------------ ---------------
  1.0 Draft   2026-07-29   Initial draft
  1.1 Draft   2026-09-04   Updated the Implementation Status note: a real Kontron SSH diagnostics integration now exists (`internal/diagnostics/kontron`), but outside the `Plugin` interface -- documented as a candidate first real Plugin rather than evidence the Capability Model is load-bearing
  1.2 Draft   2026-09-08   Documented the network-wide ONU blacklist scan added to `internal/diagnostics/kontron`, and the new write-capable `internal/provisioning/kontron` (ONU authorization) -- Palladium's first real vendor config-change command, still outside `internal/plugin`, guarded by its own RBAC capability
  1.3 Draft   2026-09-08   Corrected the Implementation Status note: `internal/provisioning/kontron/plugin` now exists and is registered -- Palladium's first real (non-simulated) `Plugin`, handling ProvisionService/ResumeService/SuspendService/DisconnectService for real over SSH. Documented that ONU authorization/deauthorization ("Discover ONU"/"Deauthorize ONU") and Remove Customer's OLT teardown call the same underlying Kontron service layer directly and are still, by design, standalone REST rather than Capability Model dispatch

------------------------------------------------------------------------

# Related Documents

-   03-DOMAIN-MODEL.md
-   05-WORKFLOW-ENGINE.md
-   07-UI-ARCHITECTURE.md

**End of Document**