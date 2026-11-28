<!--
  Licensed to the Apache Software Foundation (ASF) under one
  or more contributor license agreements.  See the NOTICE file
  distributed with this work for additional information
  regarding copyright ownership.  The ASF licenses this file
  to you under the Apache License, Version 2.0 (the
  "License"); you may not use this file except in compliance
  with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing,
  software distributed under the License is distributed on an
  "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
  KIND, either express or implied.  See the License for the
  specific language governing permissions and limitations
  under the License.
-->

# Apache Iceberg Go Security Threat Model

## Status and Reporting

This document is detailed guidance for Apache Iceberg Go maintainers and
automated security triage. It describes classification defaults and the
evidence needed to raise confidence; it does not replace case-by-case review by
the project or the Apache Security Team.

Possible vulnerabilities must be reported privately through the Apache Security
Team process. This document guides classification and scanner confidence; it
does not authorize public disclosure and does not automatically accept or
reject a report.

Follow the [Apache Iceberg security model](https://iceberg.apache.org/security/)
and the [Apache security vulnerability reporting
process](https://www.apache.org/security/). Send undisclosed Iceberg reports to
`security@iceberg.apache.org`; the Apache Security Team's
`security@apache.org` address remains a private fallback. Do not open a public
issue for a possible undisclosed vulnerability.

## Purpose

Apache Iceberg Go is a client library, table-format implementation, catalog
client, and command-line tool. It is commonly embedded in applications and
services that supply their own authorization, credential management, and
storage policy.

This threat model helps a reviewer describe:

- the actor and the capability the actor already has;
- the input or service the actor controls;
- the Iceberg Go-owned or external boundary affected;
- any secret or credential that reaches a new audience; and
- demonstrated confidentiality, integrity, availability, memory, or
  destructive impact.

The categories below are conditional triage defaults, not blanket rejection
rules. A finding that crosses an Iceberg Go-owned boundary or demonstrates a
security impact must be reviewed even when it also has a normally trusted
precondition.

## Definitions

A **new audience** is any log, error, CLI text or JSON output, serialized
metadata, host, catalog or client, or principal that was not already authorized
to receive the secret, credential, or credential-bearing request.

A **documented Iceberg Go-owned availability boundary** is an explicit
project-owned resource limit that Iceberg Go documents and enforces. It is not
a general availability guarantee for an embedding service.

Current examples include:

- the Puffin reader's configured
  [`puffin.WithMaxBlobSize`](puffin/puffin_reader.go) limit (256 MB by default);
  and
- the REST metrics reporter's documented per-report deadline, configured with
  [`rest-metrics-reporting-timeout-ms`](website/src/configuration.md#metrics-reporting)
  (10 seconds by default).

This list is non-exhaustive. An integer-width check, slice bound, worker count,
batch size, or other internal implementation constraint is not a documented
availability boundary unless a public API or project document presents it as a
resource limit and Iceberg Go enforces it.

## Scope

This model covers Iceberg Go-owned behavior in:

- the Go library and CLI;
- catalog implementations and REST catalog clients;
- configuration, HTTP transport, authentication, request signing, metrics, and
  delegated storage credentials;
- table metadata, manifests, data-file planning, deletion vectors, and Puffin
  files; and
- built-in and registered IO adapters, including local filesystem operations.

It is not a complete threat model for every process or deployment that embeds
Iceberg Go. Application user authorization, provider IAM, storage ACLs,
catalog-side credential scope, and embedding-application tenant isolation are
external enforcement points unless Iceberg Go explicitly takes ownership of a
more specific boundary.

## Security Goals

Iceberg Go should:

- prevent tokens, client secrets, storage credentials, signed requests, and
  credential-bearing configuration from reaching a new audience;
- preserve per-catalog and per-client authentication and credential isolation
  for state Iceberg Go creates, including internally managed authentication and
  transport state, REST metrics reporters and dispatchers, and
  delegated-credential state;
- avoid creating network, signing, storage, or destructive capabilities that
  the configured principal did not authorize;
- preserve documented Iceberg Go-owned availability boundaries by enforcing
  their explicit resource limits;
- avoid attacker-observable memory disclosure or memory corruption in direct
  or transitive native or `unsafe` behavior; and
- avoid deleting or mutating objects beyond the actor's proven table,
  warehouse, catalog, or storage capability.

Iceberg Go is not the primary enforcement point for application user
authorization, provider IAM, storage ACLs, catalog-side credential scope, or
tenant isolation in the embedding application. Those external responsibilities
do not waive the Iceberg Go-owned isolation and routing goals above.

## Roles

### Operator

The operator chooses catalog properties, initial endpoints, warehouse and
storage roots, transports, TLS and proxy settings, and credentials. Those
choices are trusted deployment inputs. An operator may also deliberately
install plugins or share objects between clients.

### Catalog control plane

The selected catalog resolves tables and may supply metadata, locations,
configuration, endpoint capabilities, and delegated storage credentials. The
catalog is normally trusted for those control-plane choices, but that trust
does not authorize Iceberg Go to send credentials to an unintended audience or
to cross a separate client boundary.

### REST catalog client

The REST client applies operator and catalog configuration, constructs
catalog-local sessions, routes OAuth and catalog requests, optionally signs
requests with SigV4, negotiates advertised endpoints, and selects
prefix-matching vended credentials for storage IO. Client-owned routing,
selection, caching, or reuse bugs are in scope when they create a new audience,
cross a separately constructed catalog or client, or add an unauthorized
capability.

### Embedding application

The embedding application decides which users may invoke Iceberg Go and owns
its user and tenant boundaries. It may intentionally share an `AuthManager`,
custom transport, `metrics.Reporter`, a reporter factory registered with
`metrics.Register`, database handle or implementation, registry implementation,
or other mutable dependency.
Such deliberate sharing is caller-owned; Iceberg Go-created state that crosses
otherwise separate catalog or client instances is not.

### Table writer or maintainer

A writer or maintainer may be authorized to replace metadata, add or remove
table references, write files, and invoke maintenance operations. A reviewer
must establish the actor's actual capability, the affected objects, and the
audience before treating a new path as equivalent to an authorized operation.

### CLI operator

The CLI is an operator-authorized client, not a privilege boundary.
Confirmations, `--yes`, and dry-run behavior are safety UX. Destructive
commands execute with the caller's catalog and storage authority; the CLI does
not add a separate authorization layer.

## Trust Boundaries

### Boundary 1: operator-trusted configuration

Catalog properties, initial endpoints, warehouse and storage roots, custom
transports, TLS configuration, proxy settings, and credentials are
operator-trusted deployment inputs. A finding that requires direct control of
those values normally has a trusted-input precondition and therefore lower
confidence.

That downgrade does not apply when Iceberg Go exposes a secret to a new
audience, reuses state across a separately constructed catalog or client,
bypasses an operator-selected restriction, causes attacker-observable memory
impact, or performs an unauthorized mutation or deletion.

### Boundary 2: catalog-supplied metadata and locations

The selected catalog is trusted to provide table metadata, table and file
locations, table properties, manifests, data-file references, statistics, and
other control-plane information. Iceberg Go follows those values to construct
IO and table operations; it does not impose a universal table-root sandbox.

Incorrect or malicious catalog-supplied content is normally a trusted-input or
robustness precondition. It remains security-relevant when processing it
discloses a secret, crosses client state, creates an unauthorized capability,
causes attacker-observable memory disclosure or corruption, or deletes or
mutates objects beyond the actor's proven authority.

### Boundary 3: REST configuration, routing, and delegated storage access

The selected REST catalog is trusted to supply metadata, locations,
`/v1/config` defaults and overrides, endpoint capability advertisements, and
vended storage credentials. Configuration merging may change the effective
REST base URI, OAuth route, catalog prefix, and other client properties.
Advertised capabilities select REST operations, while location-prefix matching
selects vended credentials used to construct storage IO.

A malicious control plane is normally a trusted-input precondition, but that
precondition does not dismiss a client bug that forwards a credential to a new
host, creates an unexpected outbound request, crosses a separately constructed
catalog/client boundary, or bypasses an operator-selected restriction.

OAuth routing, SigV4 signing, endpoint selection, and delegated-credential
selection therefore require impact-based review. The mere presence of a
configurable endpoint, server override, advertised endpoint, or vended
credential is not by itself a vulnerability.

### Boundary 4: storage-level authorization

Storage providers enforce object permissions through provider IAM, storage
ACLs, and the credentials made available to Iceberg Go. Catalog-side credential
scope is also an external enforcement point.

A storage operation that is already authorized against the same objects is
normally not a new Iceberg Go capability. A client bug remains in scope if it
misroutes credentials, expands their effective use, crosses client state, or
deletes or mutates objects outside the actor's proven storage capability.

### Boundary 5: Iceberg Go-owned client state

Iceberg Go internally creates REST sessions, per-catalog authentication and
transport state, REST metrics reporters and dispatchers, and
delegated-credential state. Separately constructed catalogs and clients must
not receive one another's internally managed state.

Callers may intentionally share an `AuthManager`, custom transport,
`metrics.Reporter`, a reporter factory registered with `metrics.Register`,
database handle or implementation, registry implementation, or other mutable
dependency. Effects inherent in an explicitly shared object are caller-owned by
default. This exception does not disclaim credential crossover or state reuse
caused by Iceberg Go-owned per-catalog or per-client state.

### Boundary 6: configuration discovery and process-global registries

Iceberg Go loads operator-controlled configuration from
`~/.iceberg-go.yaml`, from the directory selected by `GOICEBERG_HOME`, or from
an explicit CLI `--config` path. The library also initializes process-global
configuration state, and its catalog, IO, and metrics reporter-factory
registries are process-global plugin mechanisms that callers can extend.

Those documented configuration sources and registries are not tenant-isolation
boundaries. Control of them normally demonstrates operator or
embedding-application authority and lowers scanner confidence. Reassess when
Iceberg Go sends a secret to a new audience, crosses internally managed
catalog/client state, bypasses an operator-selected restriction, creates an
unauthorized integrity capability, exposes or corrupts memory, or causes an
unauthorized destructive effect.

### Boundary 7: table provenance

Iceberg Go follows caller- or catalog-selected metadata, manifest, data, and
Puffin locations without authenticating who authored the referenced content.
Dataset provenance and admission normally belong to the caller, catalog, or
embedding application.

Missing or malicious provenance is therefore a confidence-lowering
precondition, not a blanket exclusion. A new credential audience,
cross-catalog or cross-client state reuse, unauthorized integrity impact,
attacker-observable memory impact, or destructive side effect remains
reviewable regardless of who supplied the table.

## In-Scope Security Vulnerabilities

The following categories are higher-confidence when the report identifies a
credible actor, controlled input, affected boundary, and reproducible impact.

A demonstrated violation of a documented Iceberg Go-owned availability
boundary is also in scope. The report must identify the enforced resource limit
and show how Iceberg Go failed to enforce it; resource exhaustion with no such
violation remains hardening by default.

### 1. Secret or credential disclosure to a new audience

Examples include tokens, client secrets, storage credentials, signed requests,
or credential-bearing configuration reaching:

- a new log, error, CLI text or JSON output, or serialized metadata record;
- a host other than the one authorized to receive it;
- another catalog or client; or
- another principal.

The report should identify the secret, its intended audience, the new audience,
and the Iceberg Go-owned path that disclosed it.

### 2. Iceberg Go-owned trust-boundary violations

Internally managed authentication or transport state, REST metrics reporters
or dispatchers, or delegated-credential state crossing separately constructed
catalog or client instances is in scope. This includes unintended caching or
reuse even when both instances run in one process.

The mere existence of a process-global registry or a caller-supplied shared
object is not sufficient. The report must show that Iceberg Go-owned state
crossed a boundary the caller did not intentionally combine.

### 3. New unauthorized client capabilities

Client-owned OAuth routing, SigV4 signing, REST endpoint selection, or
vended-credential prefix selection is in scope when it creates a network,
signing, or storage capability the configured principal did not authorize.
Bypassing an operator-selected endpoint, TLS, proxy, credential-scope, or
routing restriction is also reviewable.

A configurable endpoint or a catalog-advertised operation alone is not a
vulnerability; the report must show the new capability and why the relevant
principal did not already authorize it.

### 4. Demonstrated memory disclosure or corruption

Demonstrated attacker-observable memory disclosure or memory corruption through
direct or transitive native or `unsafe` behavior is in scope. Classification
depends on observable confidentiality or integrity impact, not on whether the
failure originates in this repository or a dependency.

The mere presence of an `unsafe` zero-copy conversion, a Go panic, or malformed
input is not sufficient without the memory or boundary impact.

### 5. Unauthorized destructive effects

Deletion or mutation beyond the actor's proven table, warehouse, catalog, or
storage capability is in scope. Review cross-table or cross-root writes and
deletions, and purge behavior that reaches objects the actor was not authorized
to affect.

Iceberg Go does not guarantee universal table-root containment.
`write.data.path` and `write.metadata.path` may select paths outside the table
root, local IO operates on supplied filesystem paths, and `PurgeFiles` may
delete referenced files outside the table root. Those behaviors are not
automatically vulnerabilities, but cross-table/root deletion or unauthorized
purge remains reviewable.

## Usually Out of Scope or Non-Security by Default

The following categories usually lower confidence rather than reject a report.
Every default is conditional: a demonstrated new secret audience,
cross-catalog or cross-client effect, unauthorized integrity capability,
attacker-observable memory impact, a demonstrated violation of a documented
Iceberg Go-owned availability boundary, or unauthorized destructive effect
requires reassessment.

### 1. Correctness bugs

Incorrect metadata results, ambiguous matching, specification deviations, and
logic errors are normally correctness issues when they affect only results the
actor was already authorized to obtain or change.

Reassess when the bug exposes a secret, crosses internally managed client
state, creates an unauthorized capability or integrity effect, discloses or
corrupts memory, violates a documented Iceberg Go-owned availability boundary,
or causes an unauthorized mutation or deletion.

### 2. Parser hardening and malformed-input robustness

For malformed metadata, manifests, Avro, Parquet, Puffin, or deletion vectors,
panics, errors, allocation amplification, decompression expansion, and
availability-only failures are robustness or hardening findings by default.

Reassess when the report demonstrates secret exposure, attacker-observable
memory, unauthorized integrity impact, cross-client effects, or a demonstrated
violation of a documented Iceberg Go-owned availability boundary.

Destructive processing of referenced paths is also reviewable when it exceeds
the actor's proven authority.

### 3. Resource exhaustion and algorithmic complexity

CPU, memory, goroutine, IO, network, allocation, or decompression amplification
is normally hardening work when the only demonstrated effect is availability
within resources the caller or trusted catalog was already allowed to consume.
This includes readers that buffer an input or expand compressed data without a
decoded-size limit.

Reassess when the report demonstrates secret exposure, cross-client effects,
unauthorized integrity impact, attacker-observable memory, an unauthorized
destructive effect, or a demonstrated violation of a documented Iceberg
Go-owned availability boundary.

### 4. Malicious catalog or external service scenarios

A report that requires the selected catalog, OAuth service, storage service, or
other operator-trusted control plane to be malicious normally has a trusted
precondition and lower confidence. Trust in that service is not proof that all
resulting client behavior is authorized.

A new credential audience, unexpected outbound request, cross-catalog or
cross-client state reuse, bypass of an operator-selected restriction,
unauthorized integrity or memory impact, a demonstrated violation of a
documented Iceberg Go-owned availability boundary, or unauthorized destructive
effect overrides this default downgrade.

### 5. Equivalent-harm and authorized-writer reports

Equivalent-harm claims require evidence that the actor already had the same
capability against the same objects and audience. Writer or maintainer status
alone is not that evidence, and no universal catalog path-containment guarantee
should be assumed.

`write.data.path`, `write.metadata.path`, local IO, and `PurgeFiles` may operate
on referenced external paths. A new credential audience, cross-boundary state
reuse, greater integrity capability, attacker-observable memory impact, a
demonstrated violation of a documented Iceberg Go-owned availability boundary,
cross-table/root deletion, or unauthorized purge overrides the default
downgrade.

### 6. Caller-owned shared objects and process-global registries

Effects that follow directly from a caller deliberately sharing an
`AuthManager`, custom transport, `metrics.Reporter`, a reporter factory
registered with `metrics.Register`, database handle or implementation, registry
implementation, or other mutable dependency are normally caller-owned.
Process-global catalog, IO, and metrics reporter-factory registries are plugin
mechanisms, and their existence alone is not a credential-isolation failure.

Reassess when Iceberg Go-owned per-catalog or per-client state crosses into the
shared object unexpectedly, a secret gains a new audience, or the behavior
creates unauthorized integrity or memory impact, violates a documented Iceberg
Go-owned availability boundary, or causes unauthorized destructive impact.

### 7. Configuration loaded from documented locations

Reading `~/.iceberg-go.yaml`, using the directory selected by
`GOICEBERG_HOME`, and honoring an explicit CLI `--config` path are documented,
operator-controlled configuration behavior. A report based only on an actor
who can already modify one of those selected sources normally has lower
confidence.

Reassess when loading that configuration sends a secret to a new audience,
crosses internally managed client state, bypasses an operator-selected
restriction, causes unauthorized integrity or attacker-observable memory
impact, violates a documented Iceberg Go-owned availability boundary, or
causes an unauthorized destructive effect.

### 8. Provenance of the table being read

Iceberg Go does not authenticate authorship of caller- or catalog-selected table
metadata, manifests, data files, or Puffin files. A finding based only on
untrusted content has lower confidence when the report does not establish that
the deployment admitted that content across a boundary Iceberg Go owns. An
external file reference alone is not a vulnerability.

Reassess when the content exposes a secret to a new audience, crosses
catalog/client state, creates unauthorized integrity impact, demonstrates
attacker-observable memory disclosure or corruption, violates a documented
Iceberg Go-owned availability boundary, or triggers an unauthorized destructive
effect.

## Scanner Calibration Rules

Higher-confidence signals include:

- a secret or delegated credential reaching a new audience;
- a new unauthorized capability in Iceberg Go-owned code;
- violation of internally managed per-catalog or per-client isolation;
- a demonstrated violation of a documented Iceberg Go-owned availability
  boundary; or
- demonstrated confidentiality, integrity, memory, or unauthorized destructive
  impact.

Lower-confidence signals include:

- malformed-input or resource-exhaustion findings with availability-only impact
  and no demonstrated violation of a documented Iceberg Go-owned availability
  boundary;
- a trusted malicious control plane with no new credential audience, unexpected
  outbound capability, or cross-client effect;
- an actor with proven equivalent capability against the same objects and
  audience;
- behavior inherent in caller-owned shared objects;
- documented configuration discovery; or
- data with no established provenance and no demonstrated Iceberg Go-owned
  boundary crossing.

Scanners should report the actor, controlled input, affected boundary, existing
and new capability, intended and actual credential audience, and demonstrated
impact. They should not classify the mere presence of a configurable endpoint,
`unsafe` conversion, global registry, malformed input, or external file
reference as a vulnerability.

These signals route findings for human review; they do not automatically accept
or reject a report. Endpoint routing, credential scope, Iceberg Go-owned shared
state crossing separately constructed catalog or client instances, `unsafe`
behavior, destructive paths, and violations of documented Iceberg Go-owned
availability boundaries always require explicit human review.
