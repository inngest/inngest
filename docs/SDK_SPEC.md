# Open Source SDK Spec

# Abstract

This document presents the Open Source SDK Specification for Inngest, outlining the essential functionalities and behaviours of an SDK that communicates with an Inngest Server to provide reliable code execution on any platform. The specification covers the basic requirements of an SDK when working with Inngest services, as well as guidance towards maintaining the canonical philosophy of a highly-intuitive developer experience.

# Table of Contents

- [1](#1-introduction). Introduction
  - [1.1](#11-taxonomy). Taxonomy
  - [1.2](#12-requirements). Requirements
  - [1.3](#13-technical-definitions). Technical definitions
    - [1.3.1](#131-expression). Expression
    - [1.3.2](#132-time-string). Time String
    - [1.3.3](#133-composite-id). Composite ID
    - [1.3.4](#134-event). Event
- [2](#2-aims-of-an-sdk). Aims of an SDK
- [3](#3-environment-variables). Environment variables
  - [3.1](#31-critical-variables). Critical variables
  - [3.2](#32-optional-variables). Optional variables
- [4](#4-http). HTTP
  - [4.1](#41-headers). Headers
    - [4.1.1](#411-definitions). Definitions
    - [4.1.2](#412-requirements-when-responding-to-requests). Requirements when responding to requests
    - [4.1.3](#413-requirements-when-receiving-requests). Requirements when receiving requests
    - [4.1.4](#414-requirements-when-sending-a-request). Requirements when sending a requests
  - [4.2](#42-kinds-of-inngest-server). Kinds of Inngest Server
    - [4.2.1](#421-targeting-an-inngest-server). Targeting an Inngest Server
    - [4.2.2](#422-request-verification). Request verification
    - [4.2.3](#423-proxies-and-routing). Proxies and routing
  - [4.3](#43-sync-requests). Sync Requests
    - [4.3.1](#431-receiving-a-sync-request). Receiving a Sync Request
    - [4.3.2](#432-syncing). Syncing
    - [4.3.3](#433-handling-failure). Handling failure
    - [4.3.4](#434-handling-success). Handling success
  - [4.4](#44-call-requests). Call Requests
    - [4.4.1](#441-receiving-a-call-request). Receiving a Call Request
    - [4.4.2](#442-retrieving-the-full-payload). Retrieving the full payload
    - [4.4.3](#443-executing-the-function). Executing the Function
  - [4.5](#45-introspection-requests). Introspection Requests
- [5](#5-steps). Steps
  - [5.1](#51-reporting-steps). Reporting Steps
    - [5.1.1](#511-response-shape). Response shape
    - [5.1.2](#512-ids-and-hashing). IDs and hashing
    - [5.1.3](#513-deciding-when-to-report). Deciding when to report
  - [5.2](#52-memoizing-step-results). Memoizing Step results
    - [5.2.1](#521-finding-memoized-step-data). Finding memoized Step data
    - [5.2.2](#522-memoizing-a-step). Memoizing a Step
  - [5.3](#53-available-step-types). Available Step types
    - [5.3.1](#531-run). Run
    - [5.3.2](#532-sleep). Sleep
    - [5.3.3](#533-wait-for-event). Wait for Event
    - [5.3.4](#534-invoke). Invoke
  - [5.4](#54-recovery-and-the-stack). Recovery and the stack
  - [5.5](#55-parallelism). Parallelism
- [6](#6-middleware). Middleware
  - [6.1](#61-required-functionality). Required functionality
  - [6.2](#62-client-and-function). Client and function
  - [6.3](#63-lifecycle-methods). Lifecycle methods
    - [6.3.1](#631-function-run). Function run
    - [6.3.2](#632-event-send). Event send
  - [6.4](#64-glossary). Glossary
- [7](#7-modes). Modes

# 1. Introduction

The goal of this document is to provide a standard set of rules and guidance that can be used to create SDKs that interact with an Inngest Server and form discussions around changes when introducing new features or experiences that apply to all implementations.

An Inngest SDK is expected to satisfy the rules outlined here in order to function correctly, but this document makes no rules for the specific APIs exposed to developers when writing code to satsify these rules. Creative use of the Inngest primitives made available here is encouraged, so long as the functional expectations are met.

## 1.1. Taxonomy

In this section, key terms and concepts relevant to Inngest and Inngest SDKs are defined.

Capitalized usage of these terms means that we are referring to these definitions.

- **Inngest Server**
Inngest provides both a hosted Cloud (production) service and a Dev Server which allows local development of Inngest Functions. We refer to both as an Inngest Server. This concept is explored more in Kinds of Inngest Server [[4.2](#42-kinds-of-inngest-server)].
- **Developer**
The user of the SDK, using its APIs and interfaces to use Inngest’s platform.
- **App**
A collection of Functions.
- **Sync**
The act of informing an Inngest Server that the given Functions are exposed as an App by making a request to a particular URL.
- **Signing Key**
A key used to sign requests to and from an Inngest Server to ensure secure communication.
- **Event Key**
A key used to send events to an Inngest Server.
- **Event**
A standardized JSON object representing a single Event in your system. This concept is explored more in Event [[1.3.4](#134-event)].
- **Function**
A block of code designed to be executed in response to chosen events from Inngest. When the event occurs, an Inngest Server triggers the corresponding Function through the SDK that exposes it.
- **Step**
A retriable block of code within a Function that can be utilized as part of a larger workflow.
- **Run**
A complete execution of a Function, including all of its Steps. It may encapsulate many Call Requests and span a long period of time.
- **Sync Request**
Any `PUT` request to an SDK that should trigger a Sync.
- **Call Request**
A request from an Inngest Server to an SDK to execute a Function or Step and return the result.
- **Introspection Request**
An optional `GET` request to an SDK from anywhere that can be used as a health check. This concept is explored more in Introspection Requests [[4.5](#45-introspection-requests)].

## 1.2. Requirements

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [BCP 14, RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119).

## 1.3. Technical definitions

### 1.3.1. Expression

An Expression is a string which adheres to the [Common Expression Language](https://github.com/google/cel-spec), used in various places throughout an SDK to provide powerful expression evaluation for matching Events.

### 1.3.2. Time String

A Time String is a sequence of a decimal numbers, each with an optional fraction and unit suffix, such as `"300ms"`, or `"2h45m"`. Valid time units are `"ns"`, `"us"` (or `"µs"`), `"ms"`, `"s"`, `"m"`, `"h"`, `"d"`, and `"w"`. Units `"d"` and `"w"` are equivalent to `"24h"` and `"168h"` respectively and do not consider daylight savings time changes.

They are used to express relative time periods to an Inngest Server that doesn’t rely on the time of an SDK’s platform being accurate.

### 1.3.3. Composite ID

A Composite ID is a combination of a Function’s ID and the ID of the App that contains it, joined together with a `"-"`.

For example, if an App’s ID is `"myapp"` and a Function’s ID is `"myfn"`, the Composite ID of the Function would be `"myapp-myfn"`.

Composite IDs are used to reference Functions across App boundaries.

### 1.3.4. Event

An Event is a standardized JSON payload that represents a single event within an Inngest Server. It can be used to trigger one or more functions and can be sent from server- or client-side code.

An Event always has the following structure:

```tsx
{
  /**
   * A unique ID used to idempotently process a given event payload.
   *
   * A developer can set this when sending events to ensure that the event is
   * only processed once; if an event with the same ID is sent again, it will
   * not trigger any functions.
   */
  id: string;

  /**
   * A unique ID for the type of event. We recommend using lowercase dot
   * notation for names, prepending `prefixes/` with a slash for organization.
   *
   * e.g. `cloudwatch/alarms.triggered`, `cart/session.created`
   */
  name: string;

  /**
   * Any data pertinent to the event.
   *
   * Must be an object, in order to encourage evolving data.
   */
  data: { [key: string]: any };

  /**
   * Any user data associated with the event.
   */
  user?: { [key: string]: any };

  /**
   * An integer representing the milliseconds since the unix epoch at which this
   * event occurred.
   */
  ts: number;
}
```

When a Developer sends an event using the SDK, the SDK SHOULD allow only sending the Event with a `name`.

# 2. Aims of an SDK

An Inngest SDK provides a Developer with tools to write code that can run reliably over long time spans, surviving outages or issues within application code or dependencies.  It does this by providing a new interface for conceptualizing and expressing step functions via event-driven paradigms.

To be classified as an SDK, it MUST allow a Developer to expose an App containing Functions to an Inngest Server, which can then be executed by that server with relevant Events. The method by which these are exposed to and triggered by the Inngest Server MUST adhere to the expectations below.

This means that an SDK MUST provide a Developer with the capability to satisfy the HTTP [[4](#4-http)] and Steps [[5](#5-steps)] requirements, where the Developer SHOULD NOT have to be concerned with ensuring these are adhered to.

It is also RECOMMENDED that an SDK use the native features of a language as much as is reasonable. For example, if introducing concurrency support, prefer utilizing a language’s native concurrency paradigms to ensure a Developer needs to learn as little Inngest-specific code as possible. This then also helps all future readers of the code who do not want to be concerned with how said code is being executed, but instead what it is performing.

# 3. Environment variables

Regardless of the method of communication used or the location of the Developer’s code, it’s highly likely that the platform being used allows a Developer to specify environment variables that can be used to control aspects of the SDK and provide easy configuration without changing code.

An SDK SHOULD support the automatic reading of these variables to configure various parts of an Inngest App and the way it communicates with Inngest.

## 3.1. Critical variables

An SDK MUST support these environment variables and use them to fill the appropriate configuration internally.

- `INNGEST_EVENT_KEY`
Used to specify an Event Key to be used when sending Events to an Inngest Server. Always recommend that a Developer specifies an Event Key as an environment variable to avoid checking secrets into source control.
- `INNGEST_SIGNING_KEY`
Used to specify a Signing Key to be used when contacting the Inngest API and verifying the integrity of requests from an Inngest Server. Always recommend that a Developer specifies a Signing Key as an environment variable to avoid checking secrets into source control.
- `INNGEST_SIGNING_KEY_FALLBACK`
Used to specify a fallback Signing Key in case `INNGEST_SIGNING_KEY` is invalid. The SDK MUST attempt to use `INNGEST_SIGNING_KEY_FALLBACK` any time `INNGEST_SIGNING_KEY` fails, including but not limited to: request signature validation and sending REST API requests (e.g. fetching batch).
- `INNGEST_ENV`
MUST be used to set the `X-Inngest-Env` [[4.1.1](#411-definitions)] header when making requests to an Inngest Server, which informs the Inngest Server which environment you’re wanting to send events to.

## 3.2. Optional variables

An SDK MAY support these optional variables, as well as specify any of their own that are relevant only to that specific SDK.

- `INNGEST_LOG_LEVEL`
Recommended as an environment variable to use when debugging issues with an SDK, allowing a Developer to control the log level of the SDK’s internals.
- `INNGEST_DEV`
Recommended as a way for the user to inform an SDK that it should connect to an Inngest Dev Server, either by providing a value of `1` or an origin to use. See Targeting an Inngest Server [[4.2.1](#421-targeting-an-inngest-server)].
- `INNGEST_API_BASE_URL`
Can be used to allow a Developer to inform the SDK of where it can access the target Inngest Server’s API [[4.2](#42-kinds-of-inngest-server)]. In most situations, use of `INNGEST_DEV` will suffice, but this environment variable provides more fine-grained control.
- `INNGEST_EVENT_API_BASE_URL`
Can be used to allow a Developer to inform the SDK of where it should send Events to [[4.2](#42-kinds-of-inngest-server)]. In most situations, use of `INNGEST_DEV` will suffice, but this environment variable provides more fine-grained control.
- `INNGEST_SERVE_ORIGIN`
Because an SDK exposes an HTTP endpoint for Inngest to contact, the SDK needs to know the URL at which it exists. Sometimes this can be inferred, but it may also be useful to allow a Developer to set this manually. This environment variable sets the origin and appends the discovered path or `INNGEST_SERVE_PATH`.
- `INNGEST_SERVE_PATH`
Because an SDK exposes an HTTP endpoint for Inngest to contact, the SDK needs to know the URL at which it exists. Sometimes this can be inferred, but it may also be useful to allow a Developer to set this manually. This environment variable sets the path and prepends the discovered origin or `INNGEST_SERVE_ORIGIN`.

# 4. HTTP

Inngest can use HTTP requests to communicate with an SDK in order to perform the Syncing [[4.3](#43-sync-requests)] and Calling [[4.4](#44-call-requests)] of Functions.

The SDK MUST facilitate the Developer in exposing an HTTP endpoint for Inngest and the SDK to communicate. This endpoint MUST retain the same URL for all actions outlined below, where only the request method will change and query string parameters added.

When this document refers to Inngest communicating with an SDK, it will be through this single HTTP endpoint.

Here we outline the requirements of an SDK to function correctly as these requests are made.

## 4.1. Headers

When receiving requests from or sending requests to Inngest, headers are used frequently to show intent, provide proof of a request’s authenticity, and alter execution patterns.

This section outlines all headers used for this purpose across Inngest and SDKs.

### 4.1.1. Definitions

Here we detail all HTTP headers used by requests being made between SDKs and Inngest. Other sections will refer here if a header is required to fulfill their requirements.

- `X-Inngest-Sdk`
Represents the SDK and the version of that SDK being used for this request, in the format `[sdk_name]:v[semver_version]`, e.g. `inngest-js:v1.2.3`. It MUST be provided when sending [[4.1.4](#414-requirements-when-sending-a-request)] or responding [[4.1.2](#412-requirements-when-responding-to-requests)] to requests.

The semver version is specified based on the [Semantic Versioning 2.0.0 specification](https://semver.org/spec/v2.0.0.html), and as such all semantic versions specified there (including pre-releases) are valid. It MUST contain only a single `:` character, used to separate the SDK’s name from the semver version.

The `sdk_name` part SHOULD be the name of the GitHub repository hosting the SDK’s code. If the repository is hosted within the `inngest` organization, the organization can be omitted, e.g. `[github.com/inngest/inngest-js](http://github.com/inngest/inngest-js)` results in `inngest-js:v1.2.3`, and `[github.com/example/inngest-lib](http://github.com/example/inngest-lib)` results in `example/inngest-lib:v1.2.3`.
- `X-Inngest-Signature`
A signature of the body of a request sent by Inngest in order to prove that request’s authenticity. An SDK MUST validate a request came from Inngest by validating this signature.

The format of the signature is a query string (without the leading `?`) with two values: `t` and `s`, e.g. `t=1705586504&s=3f1c811920eb25da7fa70e3ac484e32e93f01dbbca7c9ce2365f2062a3e10c26`. `t` is the timestamp for when the request was prepared by Inngest, ensuring old-but-valid requests cannot be repeated. `s` is the actual signature, which is a hex-encoded HMAC with SHA256 of the body of the request plus the timestamp.

When receiving any request that is expected to be from Inngest, an SDK MUST validate this signature. See Requirements when receiving requests [[4.1.3](#413-requirements-when-receiving-requests)].
- `X-Inngest-Env`
Used when sending Events to an Inngest Server to indicate the Inngest Environment to send Events to. Defaults to whichever environment the Event Key being used is associated with.
- `X-Inngest-Platform`
The platform being used to host the code being invoked by an Inngest Server. Used for metrics and UIs.
- `X-Inngest-Framework`
The name of the framework being used to expose the SDK’s endpoint, such as `"lambda"` or `"nextjs"`.
- `X-Inngest-No-Retry`
MUST be set to either `true` or `false` when returning an error to an Inngest Server in response to a Call Request. See Executing the Function [[4.4.3](#443-executing-the-function)].
- `X-Inngest-Req-Version`
The execution version used when handling a Call Request. See requirements when responding to requests [[4.1.2](#412-requirements-when-responding-to-requests)].
- `Retry-After`
The standard HTTP header `Retry-After` as defined in [RFC 9110](https://datatracker.ietf.org/doc/html/rfc9110#name-retry-after), with the exception of supporting an ISO 8601 date as specified [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339) instead of an [HTTP-date](https://datatracker.ietf.org/doc/html/rfc9110#http.date). Can be used when responding to a Call Request. See Executing the Function [[4.4.3](#443-executing-the-function)].
- `X-Inngest-Server-Kind`
A header received from an Inngest Server to indicate that type of Inngest Server the request came from, either `dev` or `cloud`. Note that this MUST NOT be used to decide whether to verify requests. See Kinds of Inngest Server [[4.2](#42-kinds-of-inngest-server)].
- `X-Inngest-Expected-Server-Kind`
A header sent to an Inngest Server when Syncing to indicate which type of Inngest Server the SDK intends to contact. See Syncing [[4.3.2](#432-syncing)].

### 4.1.2. Requirements when responding to requests

When responding to a Call Request from Inngest, an SDK MUST send the following headers:

- `X-Inngest-Sdk` [[4.1.1](#411-definitions)]
Requests made or responses given to an Inngest Server without this header will be rejected.
- `X-Inngest-Req-Version` [[4.1.1](#411-definitions)]
The execution version used. MUST be `1`.

### 4.1.3. Requirements when receiving requests

When receiving a request expected to be from Inngest, an SDK MUST validate that the request originated from Inngest and is not a bad actor.

To achieve this, the `X-Inngest-Signature` [[4.1.1](#411-definitions)] header is sent by Inngest, which an SDK MUST validate is correct by adhering to the following:

1. If the SDK is intending to use the Inngest Dev Server [[4.2](#42-kinds-of-inngest-server)], the SDK SHOULD NOT attempt to verify the signature or expect the `X-Inngest-Signature` [[4.1.1](#411-definitions)] header. In this case, an SDK SHOULD warn the Developer if an `X-Inngest-Server-Kind: dev` is not received.
2. If a Signing Key has not been provided by a Developer, the SDK MUST reject the request and return a `500 Internal Server Error` response.
3. If the SDK is intending to use Inngest Cloud [[4.2](#42-kinds-of-inngest-server)] and no `X-Inngest-Signature` header has been given, the SDK MUST reject the request and return a `500 Internal Server Error` response.
4. The SDK SHOULD then extract the `t` timestamp from the `X-Inngest-Signature` header and ensure it is a time within the last 5 minutes according to the SDK’s known time.

   If this is checked and the timestamp is not within this time period, the SDK MUST reject the request and return a `500 Internal Server Error` response.

5. The SDK then MUST calculate an HMAC with SHA256 by specifying the key as the `s` value of the `X-Inngest-Signature` header with the `signkey-*-` prefix removed and hashing the body of the request followed by the timestamp `t` with hex encoding.

Note that if the raw bytes of the request body are inaccessible, the body should first be parsed using the JSON Canonicalization Scheme (JCS) as specified in [RFC 8785](https://datatracker.ietf.org/doc/html/rfc8785).

If the generated hash does not exactly match the `s` value in the `X-Inngest-Signature` header, the SDK MUST reject the requests with a `500 Internal Server Error`.

If these steps have been adhered to, the request is now deemed valid and can be used to trigger the flows that follow.

### 4.1.4. Requirements when sending a request

When sending any request to Inngest, an SDK MUST adhere to the requirements when responding to requests [[4.1.2](#412-requirements-when-responding-to-requests)].

The SDK also MUST send a bearer token authorization request header, as defined in [RFC 6750](https://datatracker.ietf.org/doc/html/rfc6750#section-2.1). The value of this bearer token should be the Signing Key, where the value following the `signkey-*-` prefix is a hex-encoded SHA256 hash of that value.

For example:

```bash
# original signing key (hex-encoded)
signkey-prod-12345678

# hashed signing key
signkey-prod-b2ed992186a5cb19f6668aade821f502c1d00970dfd0e35128d51bac4649916c

# header
Authorization: Bearer signkey-prod-b2ed992186a5cb19f6668aade821f502c1d00970dfd0e35128d51bac4649916c
```

## 4.2. Kinds of Inngest Server

Inngest provides both a hosted Cloud (production) service and a Dev Server which allows local development of Inngest Functions. We refer to both as an Inngest Server.

### 4.2.1. Targeting an Inngest Server

Both kinds of an Inngest Server may be accessible by an application at the same time, so an SDK MUST provide the Developer with an explicit method of connecting to a particular one.

Choosing to target a particular server means specifying two entrypoints:

- API origin, used for accessing the Inngest Server’s API for syncing and other comms
- Event ingestion origin, used for sending Events to the Inngest Server

It is RECOMMENDED that an SDK does this by defaulting to connecting to Inngest Cloud, which is accessible at the following:

- API Origin: `https://api.inngest.com`
- Event ingestion endpoint `https://inn.gs/e/:event_key`

If this default is followed, the SDK MUST use the `INNGEST_DEV` [[3.2](#32-optional-variables)] environment variable to allow the user to connect to a Dev Server, which can optionally also include a runtime configuration option.

The `INNGEST_DEV` environment variable may have a non-empty value, including an origin as defined in [RFC 6454](https://datatracker.ietf.org/doc/html/rfc6454), for example `http://example.com`.  If the `INNGEST_DEV` value is a valid host, the SDK should use this origin as the Dev Server URL.  Otherwise, for a non-zero `INNGEST_DEV` variable the SDK MUST attempt to connect to the following default URLs:

- API Origin: `http://localhost:8288`
- Event ingestion endpoint: `http://localhost:8288/e/:event_key`

If the `INNGEST_DEV` value is a valid [RFC 6454](https://datatracker.ietf.org/doc/html/rfc6454) origin, for example `http://example.com`, the resulting endpoints will then be used:

- API Origin: `http://example.com`
- Event ingestion endpoint: `http://example.com/e/:event_key`

### 4.2.2. Request verification

When receiving requests expected to be from an Inngest Server, an SDK MUST verify their integrity [[4.1.3](#413-requirements-when-receiving-requests)]. An Inngest Dev Server, however, MUST NOT require verification.

If the recommendations when targeting an Inngest Server [[4.2.1](#421-targeting-an-inngest-server)] are followed, then presence of the `INNGEST_DEV` environment variable (or equivalent runtime configuration option) MUST turn off signature verification.

### 4.2.3. Proxies and routing

The URL through which an SDK communicates with an Inngest Server may be indirect. In these cases, an SDK SHOULD provide the Developer with a method of setting the origin of API requests and Event ingestion.

We RECOMMEND that a Developer can use the `INNGEST_API_ORIGIN` and `INNGEST_EVENT_API_ORIGIN` environment variables [[3.2](#32-optional-variables)] for this.

## 4.3. Sync Requests

A Sync is a handshake performed between Inngest and an SDK in which the SDK informs Inngest that a set of Functions belonging to an App can be executed by Inngest at the reported URL.

It is a process that is ideally triggered automatically following every deploy by a Developer, though Inngest may also perform a Sync periodically.

A Sync is always initiated with a `PUT` request, but that initiating request can come from multiple sources, including Inngest.

```mermaid
sequenceDiagram
participant ?
participant SDK
participant Inngest

?->>+SDK: 1. PUT
SDK->>+Inngest: 2. POST<br/>Sync Endpoint
alt 3. Failure
    Inngest-->>SDK: 500
    SDK-->>?: 500
else 4. Success
    Inngest-->>-SDK: 200
    SDK-->>-?: 200
end
```

The numbered statements in the diagram above will be detailed in the following sections.

### 4.3.1. Receiving a Sync Request

When an SDK receives a `PUT` request, it is treated as the first step of a Sync. This request can come automatically from Inngest, from being triggered when using an Inngest UI, or from any `PUT` request, allowing a Developer to manually trigger a deploy by contacting their own service.

The request is not expected to have any payload. When returning data in later sections, an SDK MUST adhere to the following JSON output:

```tsx
{
	message: string;
	modified: boolean;
}
```

### 4.3.2. Syncing

Following the receipt of a sync request, an SDK MUST send a `POST` request to Inngest, targeting the Sync Endpoint of the particular Inngest service it is configured to target [[4.2](#42-kinds-of-inngest-server)].

**Method:** `POST`

**URL**

The request should be sent to the API origin [[4.2.1](#421-targeting-an-inngest-server)], with a path of `/fn/register`, e.g. `https://api.inngest.com/fn/register`.

If a `deployId` query string parameter is set on the sync request received by the SDK, the SDK MUST set the same key and value on the query string of the `POST` request to Inngest and MUST NOT include the `deployId` in the URL included in the Sync’s payload. The `deployId` is used to attribute various parts of a sync’s handshake with each other.

**Payload**

A Sync’s payload MUST be the following JSON object:

```tsx
{
  /**
   * The absolute URL that Inngest should make requests to
   * when wanting to contact the App being Synced.
   *
   * This can also contain query string parameters in order
   * to provide authorization, but MUST NOT contain a `deployId`.
   */
  url: string;

  /**
   * The string `"ping"`.
   */
  deployType: "ping";

  /**
   * The ID of App being Synced. This will be used to organize
   * Sync and the Functions within this App in the Inngest UI
   * and when being called, as well as used as the `app_id` in the
   * Composite ID for each Function. MUST NOT be an empty string.
   */
  appName: string;

  /**
   * MUST be same value as the `X-Inngest-Sdk` header, representing
   * the SDK and version of that SDK being used for this
   * particular Sync.
   */
  sdk: `${string}:v${string}`;

  /**
   * The string `"0.1"`.
   */
  v: "0.1";

  /**
   * The name of the framework being used to expose the SDK's
   * endpoint, such as `"lambda"` or `"nextjs"`. Can be omitted.
   */
  framework?: string;

  /**
   * An array of objects, each representing a Function that can
   * be executed by Inngest. Order is not important.
   */
  functions: Array<{
    /**
     * The Composite ID of the Function.
     */
    id: string;

    /**
     * May be specified as a display name for the Function in the Inngest
     * Server's UI.
     */
    name?: string;

    /**
     * An array of possible triggers for the Function, defining either
     * a cron schedule or an event name and optional expression.
     */
    triggers: Array<
      | {
          /**
           * If an event, this is the name of the Event that will trigger the
           * Function.
           */
          event: string;

          /**
           * An optional Expression can also be provided, where `event` is the Event
           * that may trigger the Function.
           */
          expression?: string;
        }
      | {
          /**
           * If a cron, this is the cron schedule that will trigger the
           * Function.
           *
           * It is defined as a unix cron string, such as `0 0 * * *` for
           * midnight every day.
           */
          cron: string;
        }
    >;

    steps: {
      step: {
        id: "step";
        name: "step";
        runtime: {
          type: "http";

          /**
           * The URL that Inngest should make requests to when wanting to
           * send a Call Request specifically to this Function.
           *
           * It MUST be the `url` defined above, with two query string
           * parameters added:
           *
           *  - `stepId` with the value `step`
           *  - `fnId` with the value of the Function's ID
           */
          url: string;
        };

        retries?: {
          /**
           * The number of times to attempt this step. If this object or
           * property isn't defined, the default is `4` (which is 3 retries).
           */
          attempts?: number;
        };
      };
    };

    /**
     * Allow the specification of an idempotency key using event data. If
     * specified, this overrides the `rateLimit` object.
     */
    idempotency?: string;

    /**
     * batchEvents specifies the batch configuration on when this function
     * should be invoked when one of the requirements is fulfilled.
     */
    batchEvents?: {
      /**
       * The maximum number of events to be consumed in one batch,
       * Currently allowed max value is 100.
       */
      maxSize: number;

      /**
       * How long to wait before invoking the function with a list of events.
       * If timeout is reached, the function will be invoked with a batch
       * even if it's not filled up to `maxSize`.
       *
       * MUST be 1s to 60s.
       */
      timeout: string;
    };

    cancel?: Array<{
      /**
       * The name of the event that should cancel the function run.
       */
      event: string;

      /**
       * The expression that must evaluate to true in order to cancel the function run. There
       * are two variables available in this expression:
       * - event, referencing the original function's event trigger
       * - async, referencing the new cancel event.
       */
      if?: string;

      /**
       * An optional timeout that the cancel is valid for.  If this isn't
       * specified, cancellation triggers are valid for up to a year or until the
       * function ends.
       *
       * The time to wait can be specified using a Time String or an ISO 8601
       * date.
       */
      timeout?: string;
    }>;

    /**
     * Rate limit workflows, only running them a given number of times (limit) per
     * period. This can optionally include a `key`, which is used to further
     * constrain throttling similar to idempotency.
     */
    rateLimit?: {
      /**
       * An optional key to use for rate limiting, similar to idempotency.
       */
      key?: string;

      /**
       * The number of times to allow the function to run per the given `period`.
       */
      limit: number;

      /**
       * The period of time to allow the function to run `limit` times.
       */
      period: TimeStr;
    };

    /**
     * Debounce delays functions for the `period` specified. If an event is sent,
     * the function will not run until at least `period` has elapsed.
     *
     * If any new events are received that match the same debounce `key`, the
     * function is rescheduled for another `period` delay, and the triggering
     * event is replaced with the latest event received.
     */
    debounce?: {
      /**
       * An optional key to use for debouncing.
       */
      key?: string;

      /**
       * The period of time to after receiving the last trigger to run the
       * function.
       */
      period: TimeStr;

      /**
       * The maximum time that a debounce can be extended before running.
       * If events are continually received within the given period, a function
       * will always run after the given timeout period.
       */
      timeout?: TimeStr;
    };

    /**
     * Configure how the priority of a function run is decided when multiple
     * functions are triggered at the same time.
     */
    priority?: {
      /**
       * An expression to use to determine the priority of a function run. The
       * expression can return a number between `-600` and `600`, where `600`
       * declares that this run should be executed before any others enqueued in
       * the last 600 seconds (10 minutes), and `-600` declares that this run
       * should be executed after any others enqueued in the last 600 seconds.
       */
      run?: string;
    };

    /**
     * Concurrency specifies a limit on the total number of concurrent steps that
     * can occur across all runs of the function.  A value of 0 (or undefined) means
     * use the maximum available concurrency.
     *
     * Specifying just a number means specifying only the concurrency limit. A
     * maximum of two concurrency options can be specified.
     */
    concurrency?:
      | number
      | Array<{
          /**
           * The concurrency limit for this option, adding a limit on how many concurrent
           * steps can execute at once.
           */
          limit: number;

          /**
           * An optional concurrency key, as an Expression.  The result of this expression is used to create new concurrency groups, or
           * sub-queues, for each function run.
           *
           * The event is passed into this expression as "event".
           *
           * Examples:
           * - `event.data.user_id`:  this evaluates to the user_id in the event.data object.
           * - `event.data.user_id + "-" + event.data.account_id`: creates a new group per user/account
           * - `"ai"`:  references a custom string
           */
          key?: string;

          /**
           * An optional scope for the concurrency group.  By default, concurrency limits are
           * scoped to functions - one function's concurrency limits do not impact other functions.
           *
           * Changing this "scope" allows concurrency limits to work across environments (eg. production
           * vs branch environments) or across your account (global).
           */
          scope?: "fn" | "env" | "account";
        }>;
  }>;
}
```

**Headers**

The SDK MUST adhere to the global header requirements when sending requests [[4.1.4](#414-requirements-when-sending-a-request)].

If an `X-Inngest-Server-Kind` header was set on the sync request received by the SDK, the SDK SHOULD forward the header’s value with a new name of `X-Inngest-Expected-Server-Kind`. This header is used to combat situations where services that a Developer understands to target Inngest Cloud can not be synced with an Inngest Dev Server and vice versa.

### 4.3.3. Handling failure

If Inngest returns a non-`200 OK` status code, the Sync has failed.

A `400 Bad Request` response indicates that either the App and Function configurations passed were invalid, or the Inngest service reached was not the one that started the deployment, for example if the Inngest Dev Server starts the deployment, but the SDK reaches out to Inngest Cloud.

In this case, the response body will be JSON with the following format:

```tsx
{
	error: string;
}
```

Any other status code may not contain the same JSON payload.

An SDK SHOULD pass this error back, closing the `PUT` request, by returning a `500 Internal Server Error` with the following JSON payload:

```tsx
{
	message: "[message_from_response_error]";
	modified: false;
}
```

### 4.3.4. Handling success

If Inngest returns a `200 OK` status code, the Sync has succeeded.

The JSON returned from this request will inform us of whether the Sync added or removed any functions or if this Sync was the same as a previous one. This can be used to log a message for a Developer to see that an App has changed. With this response, the JSON payload from Inngest will be the following:

```tsx
{
	ok: true;
	modified?: boolean;
}
```

An SDK MUST respond to the initial `PUT` request with either the `modified` value passed to it from Inngest, or `false` if none was given. For example:

```tsx
{
	message: "Successfully synced.",
	modified: true;
}
```

## 4.4. Call Requests

A Call Request is a single request from Inngest to an SDK in order to discover work that must be queued or to run blocks of user code defined in Functions or Steps.

Over the course of executing a Function that makes no use of Steps, the happy path sees Inngest execute a Function only once, where the Function’s contents are run during that Execution and the result returned to Inngest. If this Function fails by user code throwing an error or some external issue such as network issues or out-of-memory errors, Inngest will retry the Function, by executing it again for each retry.

Steps see the introduction of more complex executions, but will be discussed in the later Steps [[5](#5-steps)] section.

### 4.4.1. Receiving a Call Request

When Inngest wishes to execute a Function with an Event, it will use the URL provided to it during Syncing to contact an SDK and inform it that it wishes to run a particular Function with a given payload.

This will come in the form of a `POST` request to this endpoint. An SDK MUST adhere to the requirements when receiving a request from Inngest [[4.1.3](#413-requirements-when-receiving-requests)].

Added to the URL will be a `fnId` query string parameter, which will contain the ID of the Function that Inngest wants to execute. Similar to a Sync, this ID will be a Composite ID [[1.3.3](#133-composite-id)].

This MUST be used by an SDK to find the correct Function to call. If this Function could not be found, the SDK MUST return a `500 Internal Server Error`.

The body of the request will be a JSON payload with the following format:

```tsx
{
  /**
   * The Event that triggered this Function. If this was the result of a batch,
   * this will be the first event in the batch.
   */
  event: Event;

  /**
   * The Events that triggered this Function. If this was the result of a batch,
   * this will be all events in the batch, otherwise it will be an one-element
   * array of the `event`.
   */
  events: Array<Event>;

  /**
   * Memoized Step data that can be used to resolve previous steps.
   */
  steps: {
    /**
     * The hashed ID of the Step that finished.
     */
    [stepId: string]: /**
     * The successful result of the Step.
     */
    | {
          data: any;
        }
      /**
       * The final error of the Step if it exhausted all retries.
       */
      | {
          error: {
            name: string;
            message: string;
            stack?: string;
          };
        };
  };

  ctx: {
    /**
     * The ID of this Run of the Function.
     */
    run_id: string;

    /**
     * The current attempt number of this Run of the Function. The first attempt
     * will be `0`, the second `1`, and so on. The attempt number is incremented
     * every time the function throws an error and is retried.
     */
    attempt: number;

    /**
     * A flag to disable immediate execution of Steps, which will be set if
     * previous Steps have been planned in parallel.
     */
    disable_immediate_execution: boolean;

    /**
     * A flag to indicate that the SDK MUST use the Inngest Server's API to
     * retrieve the full payload.
     */
    use_api: boolean;

    /**
     * Contextual information for how previous steps have been memoized and
     * where this Call Request expects to be discovering Steps.
     */
    stack: {
      stack: Array<string>;
      current: number;
    };
  };
}
```

The `Event` type in the payload above refers to Inngest’s Event payload shape [[1.3.4](#134-event)].

### 4.4.2. Retrieving the full payload

Many platforms limit the incoming payload size of the functions they host. For an SDK, this means that requests with many batched Events or many memoized Steps may result in a payload that falls over the limit.

To combat this, an Inngest Server will trim large requests and instead set `ctx.use_api` in the Call Request body to `true`.

If this is set, an SDK MUST perform two requests to the Inngest API, retrieving batched Events and memoized Step data. Each request MUST adhere to the requirements when sending a request to an Inngest Server [[4.1.4](#414-requirements-when-sending-a-request)].

**Retrieving Events**

Using the API origin [[4.2.1](#421-targeting-an-inngest-server)], make a `GET` request to `[api_origin]/v0/runs/:run_id/batch`, where `run_id` can be accessed at `ctx.run_id` in the Call Request’s body.

The API will respond with either a `200 OK` with the JSON body being an array of Events [[1.3.4](#134-event)], or a non-`200 OK` response with an `{ error?: string }` JSON object.

If the request was successful, the `events` passed to the Developer’s code [[4.4.3](#443-executing-the-function)] MUST be the array returned from the endpoint, and `event` MUST be the first element of that array. If the request failed or returned a non-`200 OK` status code, the SDK MUST respond to the Call Request with a `500 Internal Server Error`.

**Retrieving memoized Step data**

Using the API origin [[4.2.1](#421-targeting-an-inngest-server)], make a `GET` request to `[api_origin]/v0/runs/:run_id/actions`, where `run_id` can be accessed at `ctx.run_id` in the Call Request’s body.

The API will respond with either a `200 OK` and the JSON body being an object equivalent to the `steps` value in a Call Request body, or a non-`200 OK` response with an `{ error?: string }` JSON object.

If the request was successful, the returned Step data should be used to memoize discovered Steps. If the request failed or returned a non-`200 OK` status code, the SDK MUST respond to the Call Request with a `500 Internal Server Error`.

### 4.4.3. Executing the Function

When an SDK has now validated the request is from Inngest and has retrieved the entire payload, it is ready to run the code represented by the Function.

An SDK SHOULD provide the Developer’s code with select fields from the Call Request’s payload, ensuring that a Developer can utilize these fields to make appropriate decisions in code and change functionality based on the situation. We RECOMMEND that these values are provided, with casing changed to the most idiomatic casing used in a particular SDK:

- `event` - The entire JSON object passed to the Execution.
- `events` - The entire array of JSON objects passed to the Execution.
- `runId` - Accessed from `ctx.run_id`.
- `attempt` - Accessed from `ctx.attempt`.
- `step` - Step tooling as detailed in Steps [[5](#5-steps)].

An SDK MUST then call the Function and return either the returned value as JSON or an error to Inngest. When responding, an SDK MUST adhere to the requirements of responding to a request from an Inngest Server [[4.1.2](#412-requirements-when-responding-to-requests)].

A successful response to Inngest MUST always return `200 OK` with a JSON payload which is the return value of the function.

If the called Function returns/throws/raises an error, the SDK MUST respond with the following JSON payload:

```tsx
{
	name: string;
	message: string;
	stack?: string;
}
```

The status code (and headers) set for the response depend on whether the Function is deemed “retriable.” An SDK SHOULD provide a Developer with a method of indicating that the error being returned/thrown/raised is “not retriable,” such that the Inngest Server will not not re-execute the Function and it will be marked as “failed.”

If this error type is available and detected by an SDK, the SDK MUST return a `400 Bad Request` status code with the `X-Inngest-No-Retry: true` header.

If this error type is not available or not detected by an SDK, the SDK MUST return a `500 Internal Server Error` status code with the `X-Inngest-No-Retry: false` header.

An SDK MAY also provide the Developer with a method of indicating how long the Inngest Server should wait before retrying, using the `Retry-After` header as defined in [RFC 9110](https://datatracker.ietf.org/doc/html/rfc9110#name-retry-after), with the exception of supporting an ISO 8601 date as specified [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339) instead of an [HTTP-date](https://datatracker.ietf.org/doc/html/rfc9110#http.date).

If `Retry-After` is set, an SDK MUST also set `X-Inngest-No-Retry: false`.

The execution of a Function, the returned data, and the tooling available to a Developer changes if Steps are introduced. See Steps [[5](#5-steps)].

## 4.5. Introspection Requests

An SDK MUST respond to an Introspection Request (a `GET` request intended to be used as a health check), ensuring that an HTTP request can be correctly handled by an SDK.

Response schemas MUST be versioned using the `schema_version` field. New schema versions MUST be added to Inngest Servers before they are used by an SDK.

The SDK MUST perform request signature validation if the `X-Inngest-Signature` header is present. If validation fails or the header is not present then the SDK MUST return the unauthenticated response (containing no privileged information). If validation succeeds, the SDK MUST return the authenticated response.

The following is schema version `2024-05-24`:

```ts
// Unauthenticated response
{
  // Will be null if the request was unsigned and false if the request was
  // signed but failed validation
  authentication_succeeded: false | null

  extra?: Record<string, any>
	function_count: number
	has_event_key: boolean
	has_signing_key: boolean
  has_signing_key_fallback: boolean
  mode: "cloud" | "dev"
  schema_version: "2024-05-24"
}

// Authenticated response
{
  api_origin: string
  app_id: string
  authentication_succeeded: true

  // Inngest environment name
  env: string | null

  extra?: Record<string, any>
  event_api_origin: string

  // sha256 hash of the event key
  event_key_hash: string | null

  framework: string
	function_count: number
	has_event_key: boolean
	has_signing_key: boolean
  has_signing_key_fallback: boolean
	mode: "cloud" | "dev"
  schema_version: "2024-05-24"
  sdk_language: string
  sdk_version: string
  serve_origin: string | null
  serve_path: string | null

  // sha256 hash of the signing key
	signing_key_fallback_hash: string | null

  // sha256 hash of the signing key fallback
	signing_key_hash: string | null
}
```

An SDK MUST NOT set any top-level keys not specified in the aforementioned schemas. An SDK MAY put any arbitrary data into the `extra` field. This avoids name collisions in the future.

# 5. Steps

Steps are a key part of an SDK, allowing a Developer to break work into multiple pieces to create step functions with an interface that allows the developer to specify it as they would any other code. Steps are dynamically discovered and executed by the Inngest Server for every Run.

When an SDK exposes and supports Steps, it will receive multiple Call Requests for the same Run, with each exercised Call Request now comprising of separate phases of first memoizing the data of previous Steps, and then executing new Steps, sometimes directed by an Inngest Server.

The Inngest Server acts as a high-level event-loop, calling an SDK to both discover and execute blocks of code. This dynamic pattern ensures that complex paradigms like concurrency and parallelism can be expressed in a Developer’s code and reliably executed across huge time spans and evolving underlying code.

Critically, to support this an SDK MUST maintain determinism in Call Requests when the underlying code has not changed, ensuring that two identical Call Requests will always produce the same output to the Inngest Server.

This example demonstrates a very simple step function, where we report, memoize, and execute Steps in separate Call Requests.

```tsx
async ({ step }) => {
	const A = await step.run("first-step", () => "A");
	const B = await step.run("second-step", () => A + "B");

	return B;
}
```

```mermaid
sequenceDiagram
autonumber
participant I as Inngest Server
participant SDK

I ->>+ SDK: POST - First Call Request (CR)<br/>state: []
note over SDK: Finds only Step A, so runs it
SDK -->>- I: 206<br/>[A: Step { data: "A" }]
I ->>+ SDK: POST - CR<br/>state: [A]
note over SDK: Memoizes Step A
note over SDK: Finds only Step B, so runs it
SDK -->>- I: 206<br/>[B: Step { data: "AB" }]
I ->>+ SDK: POST - CR<br/>state: [A,B]
note over SDK: Function completes with "AB"
SDK -->>- I: 200<br/>"AB"
```

## 5.1. Reporting Steps

When a Call Request is received and a Function is executed [[4.4.3](#443-executing-the-function)], Steps may be reported back to the Inngest Server to be separated into their own retriable Call Requests. Commonly this is done by an SDK providing a Developer with “step tooling,” which the Developer can utilize to report different step types by way of calling functions, where those function calls return data or errors based on the memoization process.

When a step is reported, depending on the Step type [[5.3](#53-available-step-types)], it will later be executed in its own Call Request or will be filled with data by an Inngest Server. As this is done, these results are then included in subsequent Call Requests, allowing the stateless SDK to memoize previous Steps and continue on to report and execute new ones. Steps that utilize these memoized results MUST NOT be reported in subsequent Call Requests.

### 5.1.1. Response shape

A Step (or multiple Steps) MUST be reported to an Inngest Server using a `206 Partial Content` response code, with a JSON payload of the following shape:

```tsx
Array<{
	id: string;
	op: string;
	displayName?: string;
	opts?: { [key: string]: any };
}>
```

Each object in the array represents a separate step being reported to the Inngest Server that will be executed in its own Call Request. Note that an SDK MUST also adhere to the requirements when sending or responding to requests [[4.1.2](#412-requirements-when-responding-to-requests)].

- `id` is the hashed ID of the Step
- `op` is the string “opcode” (an abbreviation of “operation code”) that specifies the type of Step being reported. The functionality and data expectations of each Step type is different. See Available Step types [[5.3](#53-available-step-types)].
- `displayName` is an optional string that can be given to provide a human-readable name to the Step in UIs and logging. If this is not specified by the user, it is RECOMMENDED that an SDK set this to the ID for the step that was provided instead.
- `opts` is an object used across many Step types to fine tune their functionality. For information on the expectations of this object for each Step type, see Available Step types [[5.3](#53-available-step-types)].

### 5.1.2. IDs and hashing

The `id` of each Step MUST be a hex-encoded SHA-1 hash such that the Step can be identified reliably during multiple Call Requests. Commonly, this is done by hashing a human-readable identifier provided by the Developer.

For example, the human-readable identifier `my-step-id` would be hashed as `e7d8a2f140845095749d60246ff1110c9d01d76a`.

IDs are hashed to ensure a consistent length and format across multiple SDKs, allowing cross-language, cross-cloud migrations of Functions mid-Run.

An SDK MUST also support a Developer repeating the same ID during execution. This is commonly indicative of a Developer reporting a Step within a loop, where the repeated Step is only defined once in code. To support this, each repeated human-readable identifier MUST append `:n` to the end of the identifier before hashing, where `n` is the number of times the Step has previously been found.

For example, if `my-step-id` is repeated 3 times, the pre-hash identifiers for those steps should be `my-step-id`, `my-step-id:1`, and `my-step-id:2`. Note that the zero-index item MUST have no `:n` suffix.

### 5.1.3. Deciding when to report

When running the Developer’s code within a Function, an SDK must decide when to halt execution and report Steps back by responding to the Call Request. Commonly, this is done by waiting until a new Step is found that has no memoized result [[5.2](#52-memoizing-step-results)].

Parallelism [[5.5](#55-parallelism)] introduces more complexity to this.

## 5.2. Memoizing Step results

The process of memoizing Step results is receiving data or errors that the Inngest Server passes in a Call Request to return that data or error to a Developer’s code, such that they can continue to report more steps or end the function by returning a value or an error.

### 5.2.1. Finding memoized Step data

When a Call Request is made to an SDK, it can contain memoized data from previously-run Steps in the `steps` object passed, which is an object where the keys are hashed Step IDs and the value memoized data that changes shape based on the op code being used.

If an object with ONLY a `data` key is found, its value will be the result of that particular Step type [[5.3](#53-available-step-types)], serialized as JSON. An SDK SHOULD make a Developer aware of this serialization process, as first-class entities such as class instances may be lost.

If an object with ONLY an `error` key is found, its value will be the final error returned when attempting to run the Step, in the same error format used for reporting function failures:

```tsx
{
	name: string;
	message: string;
	stack?: string;
}
```

Other values such as `null` can also be found depending on the Step type [[5.3](#53-available-step-types)], where the particular Step type being used indicates its meaning.

### 5.2.2. Memoizing a Step

As Steps are found by an SDK during a Call Request, it MUST try to find either the successful data already returned for that Step or the error that caused it to fail by matching the hashed Step ID with a key in the `steps` object. If this is found, the SDK MUST either return the relevant data to the Developer’s call or return/throw/raise the error.

If an `error` is returned/thrown/raised and the Developer has not either swallowed the error or returned/thrown/raised a new error, the SDK MUST mark the request as non-retriable [[4.4.3](#443-executing-the-function)]. When this error is being serialized to be sent back to an Inngest Server, it SHOULD be the original serialized error that was returned/thrown/raised.

If a Step’s hash is present in the `steps` object passed in the Call Request, it MUST NOT be reported when responding to that Call Request; only new Steps without a memoized `data` or `error` can be reported.

If a Step is found to have a memoized `error`, an SDK SHOULD create the equivalent of a `StepError` value, passed to the Developer such that they can use it to access the `name`, `message`, and optional `stack` property.

The `StepError` value SHOULD aim to be used as a language-native error, e.g. JavaScript should extend `Error`, Python should extend `Exception`, and Go should implement the `error` interface. This ensures that the Developer can utilize language features such as `error.As()` or `except StepError as e` to appropriately catch and type the error.

## 5.3. Available Step types

There are various Step types available that an SDK can report and handle, each providing a small building block on which to build complex event-driven functionality.

An SDK SHOULD name these concepts as they are named here, and SHOULD make access to them obvious, for example via exposing `step.run()` for a Run Step, or `step.sleep()` for a Sleep Step.

An SDK MAY alias Step types to fit in with the language and implementation being provided, for example by providing `step.sleepForSeconds()` to avoid polymorphism. In this case, the Steps still MUST be obvious enough in their naming that a Developer understands explicitly which Step type is being used.

An SDK MAY also abstract the underlying Steps being used, but the underlying Steps MUST be documented such that a Developer can easily understand and discover how many Steps will run as a result of their function calls, in order to aid in billing transparency.

### 5.3.1. Run

The Run Step is the simplest Step type. The Developer specifies a block of code to run and the Inngest Server makes a separate Call Request to run only that piece of code.

This Step type encapsulates multiple “op codes”, used by both the SDK and the Inngest Server to express the intent and status of the Run Step.

When finding the Step, an SDK can choose to either always report the Step [[5.1](#51-reporting-steps)] or to immediately execute it. Immediate execution MUST only be allowed if `ctx.disable_immediate_execution` is `false` in the Call Request body, the query string parameter `stepId` is `"step"` in the Call Request, and the SDK only found a single Step to report which is a Run Step.

If these conditions are satisfied, an SDK can either run the Developer’s code within the same Call Request and then return a `Step` operation.

```tsx
{
	id: string;
	op: "StepRun";
	data: any; // the serialized JSON returned from the Developer's code.
	displayName?: string;
}
```

Or the SDK can choose to first *plan* the Step by returning a `StepPlanned` operation, where the Inngest Server will create a separate Call Request to run the Developer’s code:

```tsx
{
	id: string;
	op: "StepPlanned";
	displayName?: string;
}
```

If an SDK sends a `"StepPlanned"` operation, the Inngest Server will send a separate Call Request to run the Developer’s code represented within this Step. To do this, the Inngest Server will send a `stepId` query string parameter, which is the hashed ID of the Step to run.

If this query parameter is present and NOT `"step"`, the SDK MUST NOT immediately execute or report any other Steps, and instead MUST search for the Step to be run while memoizing previous Steps.

If this Step is found, the SDK MUST run the Developer’s code represented by this Step and return either the resulting serialized JSON in the `data` key:

```tsx
{
	id: string;
	op: "StepRun";
	data: any; // the serialized JSON returned from the Developer's code
	displayName?: string;
}
```

Or the resulting error as serialized JSON:

```tsx
{
	id: string;
	op: "StepError";
	error: {
		name: string;
		message: string;
		stack?: string;
	};
	displayName?: string;
}
```

It is possible that the Step could not be found due to the underlying code changing between reporting the Step and executing it, or a non-deterministic function inserting arbitrary delays between Steps. In either case, if the Step cannot be found within a reasonable period, an SDK MUST return a `"StepNotFound"` operation:

```tsx
{
	id: string;
	op: "StepNotFound";
}
```

The memoized result of the Step will be either a `{ data }` or an `{ error }` object, depending on if the Step succeeded or failed.

### 5.3.2. Sleep

A Sleep Step informs the Inngest Server that the Run wishes to be called again after the given amount of time, specified either as an ISO 8601 date as specified [RFC 3339](https://datatracker.ietf.org/doc/html/rfc3339), or a “time string.” The latter “time string” is a sequence of decimal numbers, each with an optional fraction and unit suffix, such as `"300ms"`, or `"2h45m"`. Valid time units are `"ns"`, `"us"` (or `"µs"`), `"ms"`, `"s"`, `"m"`, `"h"`, `"d"`, and `"w"`.

When reporting the Step, the SDK MUST use the following schema:

```tsx
{
	id: string;
	op: "Sleep";
	opts: {
		duration: "[time_string_or_stringified_ISO_date]";
	};
	displayName?: string;
}
```

When the given time has elapsed, the Inngest Server will memoize the Step with `null`.

### 5.3.3. Wait for Event

A Wait For Event Step informs the Inngest Server that the Run wishes to be called again once a given event has been received. This can be defined using the name of the event, as well as an Expression [[1.3.1](#131-expression)].

```tsx
{
	id: string;
	op: "WaitForEvent";
	opts: {
		event: string;
		timeout: "[time_string]";
		if?: "[cel_expression]";
	};
	displayName?: string;
}
```

In the `if` Expression, the `event` variable represents the `event` passed in the Call Request payload. If multiple `events` have been sent, the `event` variable represents the first event in the array. The `async` variable represents the event being matched against.

When the event is received by the Inngest Server, the Step will be memoized with the entire event payload. Note that it will not be nested in `data`. If the timeout has elapsed without the Inngest Server receiving the event, the Step will be memoized with `null`.

### 5.3.4. Invoke

An Invocation Step informs the Inngest Server that the Run wishes to trigger another Inngest function and wait for its response. This provides the Developer with a method of composing Functions together.

- `opts.function_id` represents the Function to invoke, referenced by its Composite ID [[1.3.3](#133-composite-id)]
- `opts.payload` is the Event that will be sent to the function to trigger it, omitting the `name`

```tsx
{
	id: string;
	op: "InvokeFunction";
	opts: {
		function_id: string;
		payload: Event;
	};
	displayName?: string;
}
```

When the invoked Function has run to completion and returned a value, the Inngest Server will memoize the Step with either a `{ data }` or an `{ error }` object depending on whether the invoked Function succeeded or failed.

## 5.4. Recovery and the stack

When memoizing Steps [[5.2](#52-memoizing-step-results)], the Call Request will provide an array of Step IDs at `ctx.stack.stack` which represents the order in which previous Steps were completed. Each ID present will exist as a key in the `steps` object with some memoized data. This ordering can be critical if code relies on assessing race conditions, as the order in which Steps are discovered dynamically by an SDK can differ from the order in which they should be memoized.

It is therefore RECOMMENDED that an SDK follows this order as they memoize Steps and makes an effort to not just immediately memoize Steps as they are discovered.

If this recommendation is followed, the SDK can now detect some of the symptoms of a Function’s underlying code changing mid-Run, as the order defined in `ctx.stack.stack` will not be achievable.

If the SDK is following this ordering and the next Step cannot be found, it MUST first warn the user that the Function has appeared to change. Note that this symptom may also be experienced if a user’s function is non-deterministic, such that repeated executions result in different reported Steps. This should be treated as an issue that the Developer needs to address.

Next, the SDK MUST continue to memoize Steps, but no longer expect them to sequentially match the `ctx.stack.stack` ordering. Instead, upon finding each batch of Steps to attempt to memoize, the SDK MUST step over the `ctx.stack.stack` array and find the earliest found Step that should be memoized. This process is repeated until all state has been used, at which point the SDK can report or execute new Steps.

## 5.5. Parallelism

An SDK MAY support parallelism. Multiple steps can be reported in a single Call Request, where those Steps will be run in parallel in their own separate Call Requests.

If an SDK supports parallelism, it MUST follow the `ctx.stack.stack` and recovery techniques [[5.4](#54-recovery-and-the-stack)]. When a Run reports >1 Step in response to a Call Request, `ctx.disable_immediate_execution` will be `true` for all subsequent Call Requests, meaning the SDK MUST then always report Steps with `StepPlanned` before executing them [[5.3.1](#531-run)].

# 6. Middleware

Middleware allows users to run arbitrary code at certain points in the execution lifecycle. Middleware is used to add functionality such as logging, error handling, and more.

## 6.1. Required functionality

Since SDKs strive to be idiomatic to their respective languages, the ergonomics of middleware and names/signatures of lifecycle methods MAY deviate significantly between SDKs. However, all SDKs MUST support the same basic required functionality:

- Adding custom fields to the Inngest Function context (e.g. database connection)
- Accessing the framework/platform level HTTP request (i.e. before the `serve` function normalizes it)
- Mutate the memoized step data before calling an Inngest Function
- Mutate the step/function output and errors before sending to the Inngest Server
- Mutate events before sending to the Inngest Server

## 6.2. Client and function

Middleware MAY be specified on the Inngest Client and/or individual Inngest Functions:
- Client middleware is executed when any function runs.
- Function middleware is executed only when its function runs.

Client-level middleware MUST always be run before function-level middleware, and all middleware MUST be run in the order it was registered.

## 6.3. Lifecycle methods

There are two lifecycles groups to cover: a function run and sending an event.

### 6.3.1. Function run

- For a single SDK request, all lifecycle methods MUST be called EXACTLY once.
- The lifecycle methods MUST be called in the following order:
  - Transform input
  - Before memoization
  - After memoization
  - Before execution
  - After execution
  - Transform output
  - Before response

#### Transform input
- MUST call before executing the Inngest Function callback.
- Arguments:
  - Context object that's passed to the Inngest Function callback.
  - Map of memoized step data

#### Before memoization
- MUST call before beginning first memoized step call.
- Effectively the same as "transform input", but executed after it.

#### After memoization
- MUST call after last memoized step call.
- Called immediately after the "before memoization" hook if the run has no state, as we will immediately be executing new code.

#### Before execution
- MUST call before executing unmemoized code.
- MUST be called before function-level code will run that is after the last step to memoize.

#### After execution
- MUST call after executing unmemoized code.

#### Transform output
- MUST call after a function/step returns or throws an error.
- Arguments:
  - Thrown/returned error. Will not exist if the function/step returned successfully.
  - Data returned by the function/step. Will not exist if the function/step errored.
  - Step data (user-specified ID, opcode, etc.). Will not exist of a new step was not executed.

#### Before response
- MUST call after the output has been set and before the response is sent back to an Inngest Server.

### 6.3.2. Event send

- For a single request to send an event, all lifecycle methods MUST be called EXACTLY once.
- The lifecycle methods MUST be called in the following order:
  - Transform input
  - Transform output
- This entire lifecycle may be called multiple times for a single SDK request.

#### Before send events
- MUST call before sending events to an Inngest Server.
- Arguments:
  - Array of events to send.
- If a events are sent multiple times within a single SDK request, this method MUST be called before each send.

#### After send events
- MUST call after sending events to an Inngest Server.
- Arguments:
  - Array of IDs for the sent events.
  - Error if 1 or more events failed to send. Errors are "partial": it's possible to successfully send 1 event but error on another. Will not exist if all events were sent successfully.
- If a events are sent multiple times within a single SDK request, this method MUST be called before each send.

## 6.4. Glossary

### Unmemoized code

The vast majority of the time, "unmemoized code" is function-level code encountered after exhausting the memoized step data.

However, a non-deterministic function may cause the SDK to encounter an unmemoized step before exhausting the memoized step data. In this case, the unmemoized step is the start of the unmemoized code.

# 7. Modes

The SDK can be in one of 2 modes: `cloud` or `dev`. The SDK defaults to `cloud` mode, but can be switched to `dev` mode by setting the `INNGEST_DEV` environment variable.

When in `cloud` mode:
- The SDK defaults to Inngest Cloud URLs (API is `https://api.inngest.com` and Event API is `https://inn.gs`).
- All incoming request signatures are required and verified (unless stated otherwise in this spec).

When in `dev` mode:
- The SDK defaults to the Dev Server's default URL (`http://localhost:8288`).
- Incoming request signatures are only validated if present and the SDK has a signing key.
