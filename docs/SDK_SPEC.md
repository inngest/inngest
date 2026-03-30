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
    - [4.3.5](#435-in-band-sync). In-Band Sync
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
    - [5.3.5](#535-send-event). Send Event
    - [5.3.6](#536-ai-gateway). AI Gateway
    - [5.3.7](#537-gateway-http-fetch). Gateway (HTTP Fetch)
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
- [8](#8-connect). Connect
  - [8.1](#81-environment-variables). Environment variables
  - [8.2](#82-runtime-type). Runtime type
- [9](#9-streaming). Streaming
- [10](#10-checkpointing). Checkpointing
  - [10.1](#101-configuration). Configuration
  - [10.2](#102-sync-and-async-opcodes). Sync and async opcodes
  - [10.3](#103-checkpoint-api). Checkpoint API
  - [10.4](#104-execution-flow). Execution flow
- [11](#11-failure-handlers). Failure Handlers
