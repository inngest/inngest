# Inngest

![Latest release](https://img.shields.io/github/v/release/inngest/inngest?include_prereleases&sort=semver)
![Test Status](https://img.shields.io/github/workflow/status/inngest/inngest/Go/main?label=tests)
![Discord](https://img.shields.io/discord/842170679536517141?label=discord)
![Twitter Follow](https://img.shields.io/twitter/follow/inngest?style=social)

Inngest is an open-source, event-driven platform which makes it easy for developers to build, test, and deploy serverless functions without worrying about infrastructure, queues, or stateful services.

Using Inngest, you can write and deploy serverless step functions which are triggered by events without writing any boilerplate code or infra. Learn more at https://www.inngest.com.

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Project Architecture](#project-architecture)
- [Community](#community)

<br />

## Overview

Inngest makes it simple for you to write delayed or background jobs by triggering functions from events — decoupling your code from your application.

- You send events from your application via HTTP (or via third party webhooks, e.g. Stripe)
- Inngest runs your serverless functions that are configured to be triggered by those events, either immediately, or delayed.

Inngest abstracts the complex parts of building a robust, reliable, and scalable architecture away from you so you can focus on writing amazing code and building applications for your users.

We created Inngest to bring the benefits of event-driven systems to all developers, without having to write any code themselves. We believe that:

- Event-driven systems should be _easy_ to build and adopt
- Event-driven systems are better than regular, procedural systems and queues
- Developer experience matters
- Serverless scheduling enables scalable, reliable systems that are both cheaper and better for compliance

[Read more about our vision and why this project exists](https://www.inngest.com/blog/open-source-event-driven-queue)

<br />

## Quick Start

1. Install the Inngest CLI to get started:

```bash
curl -sfL https://cli.inngest.com/install.sh | sh \
  && sudo mv ./inngest /usr/local/bin/inngest
# or via npm
npm install -g inngest-cli
```

2.  Create a new function. It will prompt you to select a programming language and what event will trigger your function. Optionally use the `--trigger` flag to specify the event name:

```shell
inngest init --trigger demo/event.sent
```

3. Run your new hello world function with dummy data:

```shell
inngest run
```

4. Run the Inngest DevServer. This starts a local "Event API" which can receive events. When events are received, functions with matching triggers will automatically be run. Optionally use the `-p` flag to specify the port for the Event API.

```shell
inngest dev -p 9999
```

5. Send events to the DevServer. Send right from your application using HTTP + JSON or simply, as a curl with a dummy key of `KEY`.

```shell
curl -X POST --data '{"name":"demo/event.sent","data":{"test":true}}' http://127.0.0.1:9999/e/KEY
```

That's it - your hello world function should run automatically! When you `inngest deploy` your function to Inngest Cloud or your self-hosted Inngest. Here are some more resources to get you going:

- [Full Quick Start Guide](https://www.inngest.com/docs/quick-start?ref=github)
- [Function arguments & responses](https://www.inngest.com/docs/functions/function-input-and-output?ref=github)
- [Sending Events to Inngest](https://www.inngest.com/docs/event-format-and-structure?ref=github)
- [Inngest Cloud: Managing Secrets](https://www.inngest.com/docs/cloud/managing-secrets?ref=github)
- [Self-hosting Inngest](https://www.inngest.com/docs/self-hosting?ref=github)

<br />

## Project Architecture

Fundamentally, there are two core pieces to Inngest: _events_ and _functions_. Functions have several sub-components for managing complex functionality (eg. steps, edges, triggers), but high level an event triggers a function, much like you schedule a job via an RPC call to a queue. Except, in Inngest, **functions are declarative**. They specify which events they react to, their schedules and delays, and the steps in their sequence.

<br />

<p align="center">
  <img src=".github/assets/architecture-0.5.0.png" alt="Open Source Architecture" width="660" />
</p>

Inngest's architecture is made up of 6 core components:

- **Event API** receives events from clients through a simple POST request, pushing them to the **message queue**.
- **Event Stream** acts as a buffer between the **API** and the **Runner**, buffering incoming messages to ensure QoS before passing messages to be executed.<br />
- A **Runner** coordinates the execution of functions and a specific run’s **State**. When a new function execution is required, this schedules running the function’s steps from the trigger via the **executor.** Upon each step’s completion, this schedules execution of subsequent steps via iterating through the function’s **Edges.**
- **Executor** manages executing the individual steps of a function, via _drivers_ for each step’s runtime. It loads the specific code to execute via the **DataStore.** It also interfaces over the **State** store to save action data as each finishes or fails.
  - **Drivers** run the specific action code for a step, eg. within Docker or WASM. This allows us to support a variety of runtimes.
- **State** stores data about events and given function runs, including the outputs and errors of individual actions, and what’s enqueued for the future.
- **DataStore** stores persisted system data including Functions and Actions version metadata.
- **Core API** is the main interface for writing to the DataStore. The CLI uses this to deploy new funtions and manage other key resources.

And, in this CLI:

- The **DevServer** combines all of the components and basic drivers for each into a single system which loads all functions on disk, handles incoming events via the API and executes functions, all returning a readable output to the developer. (_Note - the DevServer does not run a Core API as functions are loaded directly from disk_)

To learn how these components all work together, [check out the in-depth architecture doc](To learn how these components all work together, [check out the in-depth architecture doc](/docs/ARCHITECTURE.md). For specific information on how the DevServer works and how it compares to production [read this doc](/docs/DEVSERVER_ARCHITECTURE.md).
).

<br />

## Community

- [**Join our online community for support, to give us feedback, or chat with us**](https://www.inngest.com/discord).
- [Post a question or idea to our Github discussion board](https://github.com/orgs/inngest/discussions)
- [Read the documentation](https://www.inngest.com/docs)
- [Explore our public roadmap](https://github.com/orgs/inngest/projects/1/)
- [Follow us on Twitter](https://twitter.com/inngest)
- [Join our mailing list](https://www.inngest.com/mailing-list) for release notes and project updates

## Contributing

We’re excited to embrace the community! We’re happy for any and all contributions, whether they’re feature requests, ideas, bug reports, or PRs. While we’re open source, we don’t have expectations that people do our work for us — so any contributions are indeed very much appreciated. Feel free to hack on anything and submit a PR.
