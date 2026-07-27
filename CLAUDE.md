# Palladium OSS

## Purpose

Palladium OSS is a modern Operations Support System for broadband providers.

Its responsibilities include:

- Inventory
- Customers
- Services
- Provisioning
- Workflow automation
- Network discovery
- Plugin management
- Audit history

Palladium is NOT:

- a monitoring platform
- an NMS
- an alerting platform
- a time-series database

Monitoring belongs in Zabbix or other monitoring systems.

---

# Core Philosophy

Always model the real world.

Customers own Services.

Services consume Resources.

Resources exist independently of Customers.

Never couple inventory directly to customers.

---

# Architecture

Backend:

Go

Frontend:

Vue 3
TypeScript
Vite

Database:

PostgreSQL

Communication:

REST API

Long-running work:

Asynchronous workers

---

# Code Quality

Prefer readability over cleverness.

Avoid unnecessary abstractions.

Small packages.

Small files.

Small functions.

Write tests.

Never duplicate business logic.

---

# Database Rules

Every schema change uses migrations.

Never modify production data directly.

Use UUID primary keys.

Prefer explicit foreign keys.

Soft-delete when historical records matter.

---

# Plugin Philosophy

Everything vendor-specific belongs in plugins.

The core system must never contain Kontron-, Nokia-, Calix-, Adtran-, MikroTik-, or vendor-specific logic.

Plugins expose capabilities.

Core orchestrates workflows.

---

# Workflow Philosophy

Everything significant should be a workflow.

Provisioning

Suspend

Restore

Replace ONU

Replace OLT

Move customer

Activate service

Deactivate service

Everything should be resumable.

Everything should be auditable.

---

# API Guidelines

REST first.

JSON only.

Consistent naming.

Version APIs.

Return structured errors.

---

# Frontend Guidelines

Composition API.

TypeScript everywhere.

Reusable components.

Accessibility matters.

---

# Naming

Use descriptive names.

Avoid abbreviations.

Avoid one-letter variables.

---

# Documentation

Every exported function should be documented.

Complex business rules deserve comments.

Code should explain HOW.

Documentation explains WHY.

---

# General Rule

When uncertain, choose the solution that is:

- simpler
- more maintainable
- easier to understand
- easier to test
