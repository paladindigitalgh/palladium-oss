# Palladium OSS Architecture

**Version:** 1.0  
**Status:** Living Document

---

# Vision

Palladium OSS is a modern, vendor-neutral Operations Support System (OSS) for broadband providers, municipalities, electric cooperatives, wireless ISPs, and fiber operators.

The system is designed to replace fragmented spreadsheets, vendor-specific management tools, and custom automation scripts with a cohesive operational platform that models the real-world network and automates operational workflows.

Palladium is designed to scale from small regional providers to large multi-market operators without requiring architectural changes.

---

# Design Goals

Palladium should be:

- Vendor neutral
- Workflow driven
- API first
- Plugin extensible
- Horizontally scalable
- Auditable
- Highly observable
- Secure by default
- Easy to automate
- Pleasant to operate

---

# What Palladium Is

Palladium manages:

- Physical inventory
- Logical inventory
- Customers
- Services
- Provisioning
- Workflow orchestration
- Device discovery
- Automation
- Network documentation
- Change history
- User permissions
- Plugin management

---

# What Palladium Is Not

Palladium intentionally does not replace specialized systems.

Monitoring belongs in:

- Zabbix
- LibreNMS
- Prometheus

Device management belongs in:

- GenieACS
- Vendor management systems

Billing belongs in:

- Sonar
- Splynx
- Custom billing systems

Palladium integrates with those systems rather than replacing them.

---

# Core Philosophy

Everything in Palladium should model reality.

Real-world objects exist independently of customers.

Examples:

An ONU exists before it is assigned.

A splitter exists whether or not it serves customers.

A fiber cable exists whether it is active.

A service exists independently of the customer consuming it.

Inventory should never be created solely because a customer exists.

---

# System Context

```
Internet

↓

Core Router

↓

Aggregation

↓

OLT

↓

PON

↓

Splitter

↓

Drop

↓

ONU

↓

Customer Equipment
```

Palladium models every layer of this hierarchy.

---

# Major Domains

## Inventory

Responsible for all physical assets.

Hierarchy: Site → Building → Room → Rack → Device.

Device is deliberately generic and vendor-agnostic — it carries no
type/subtype distinction. Vendor- and function-specific equipment
(OLTs, switches, routers, splitters, fiber, ONUs, ...) either belongs
to its own domain (see Network below) or to a future, more specific
model layered on top of Device.

Inventory exists independently of customers.

---

## Customers

Responsible for:

- Customer records
- Contacts
- Addresses
- Billing references
- Service relationships

---

## Services

Represents products delivered to customers.

Examples:

- Internet
- Voice
- IPTV
- Dark Fiber
- Enterprise Ethernet

Services consume inventory.

---

## Network

Responsible for:

- Access Networks
- OLTs
- PON ports
- Access Interfaces
- Access Attachments (linking equipment to an Access Interface)
- VLANs (planned)
- IP pools (planned)
- VRFs (planned)
- Routing relationships (planned)

---

## Workflow Engine

Every significant operational task is represented as a workflow.

Examples:

- Activate customer
- Suspend service
- Restore service
- Replace ONU
- Replace OLT
- Migrate customer
- Change package
- Discover devices

Every workflow is resumable.

Every workflow is auditable.

Every workflow is replayable when practical.

---

## Plugins

Vendor-specific functionality is implemented entirely through plugins.

Core should never contain vendor-specific logic.

Supported plugin types include:

- OLT vendors
- Router vendors
- ACS systems
- CRM systems
- Billing systems
- Monitoring systems
- Authentication providers

Plugins expose capabilities.

Core orchestrates them.

---

# Architecture

```
Vue Frontend

↓

REST API

↓

Application Services

↓

Workflow Engine

↓

Plugin Registry

↓

Vendor Plugins

↓

External Systems
```

---

# Security Model

Authentication is delegated to identity providers.

Authorization is role-based.

Future enhancements should support:

- OIDC
- LDAP
- Active Directory
- Azure Entra ID

Every action should be attributable to a user or automation account.

---

# Audit Philosophy

Everything important should be auditable.

Examples:

Inventory movement

Provisioning

Authentication

Configuration changes

Workflow execution

Plugin actions

Audit history should be immutable.

---

# Inventory Philosophy

Inventory has a lifecycle.

Example:

Ordered

↓

Received

↓

Stored

↓

Installed

↓

Provisioned

↓

Assigned

↓

Retired

↓

Disposed

Palladium should preserve the history of every transition.

---

# Customer Philosophy

Customers own services.

Services consume resources.

Resources never belong directly to customers.

---

# Data Philosophy

Avoid duplication.

Prefer references.

Prefer immutable history.

Favor normalization over convenience.

Denormalize only after measurement proves necessary.

---

# API Philosophy

REST first.

JSON only.

Version all public APIs.

Every endpoint should be documented.

All responses should be structured.

---

# Plugin Philosophy

Plugins communicate through well-defined interfaces.

Plugins should never bypass the core business logic.

Plugins should be independently testable.

Plugins should declare capabilities.

Core decides which plugin performs work.

---

# Error Handling

Errors should be:

- actionable
- descriptive
- structured
- logged
- traceable

Never hide underlying causes.

---

# Scalability

The architecture should support:

- Multiple headends
- Multiple regions
- Multiple organizations
- Multiple vendors
- Multiple provisioning workers

Horizontal scaling should require no application redesign.

---

# Technology Stack

Backend

- Go

Frontend

- Vue 3
- TypeScript

Database

- PostgreSQL

Authentication

- JWT (OIDC planned — see Security Model)

Deployment

- Docker
- Kubernetes (future)

---

# Long-Term Vision

Palladium should become the operational source of truth for a service provider.

Every operational task should either:

- model the network,
- automate the network,
- document the network,
- or explain the network.

Anything outside those goals belongs in another specialized system.
