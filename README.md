# [Inngest](https://www.inngest.com)

[![Latest release](https://img.shields.io/github/v/release/inngest/inngest?include_prereleases&sort=semver)](https://github.com/inngest/inngest/releases)
[![Test Status](https://img.shields.io/github/actions/workflow/status/inngest/inngest/go.yaml?branch=main&label=tests)](https://github.com/inngest/inngest/actions?query=branch%3Amain)
[![Discord](https://img.shields.io/discord/842170679536517141?label=discord)](https://www.inngest.com/discord)
[![Twitter Follow](https://img.shields.io/twitter/follow/inngest?style=social)](https://twitter.com/inngest)

[Inngest](https://www.inngest.com) is the developer platform for easily building reliable workflows with zero infrastructure.

<div align="center">

  <a href="https://www.inngest.com/uses/serverless-node-background-jobs?ref=org-readme">
    Background Jobs
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/uses/serverless-queues?ref=org-readme">
    Serverless Queues
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/docs/functions/multi-step?ref=org-readme">
    Workflows
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/uses/zero-infra-llm-ai?ref=org-readme">
    AI & LLM Chains
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/uses/serverless-cron-jobs?ref=org-readme">
    Scheduled Jobs
  </a>
</div>
<br/>

- Write background jobs and workflows in your existing codebase using the [**Inngest SDK**](https://github.com/inngest/inngest-js)
- Run the open source [**Inngest Dev Server**](#the-inngest-dev-server) on your machine for a complete local development experience, with production parity.
- The **Inngest Platform** invokes your code wherever you host it, via HTTPS. Deploy to your existing setup, and deliver products faster without managing infrastructure.

---

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Project Architecture](#project-architecture)
- [Community](#community)

<br />

#### The Inngest Dev Server

```
npx inngest-cli@latest dev
```

![Inngest Dev Server screenshot](https://www.inngest.com/assets/homepage/dev-server-screenshot.jpg)

<br />

## Overview

Inngest makes it easy to develop serverless workflows in your existing codebase, without any new infrastructure. Inngest Functions are triggered via events &mdash; decoupling your code within your application.

1. You define your Inngest functions using the [Inngest SDK](https://github.com/inngest/inngest-js) and serve them through a [simple API endpoint](https://www.inngest.com/docs/sdk/serve?ref=github-inngest-readme).
2. Inngest automatically invokes your functions via HTTPS whenever you send events from your application.

Inngest abstracts the complex parts of building a robust, reliable, and scalable architecture away from you, so you can focus on writing amazing code and building applications for your users.

- **Run your code anywhere** - We call you via HTTPS so you can deploy your code to serverless, servers or the edge.
- **Zero-infrastructure required** - No queues or workers to configure or manage &mdash; just write code and Inngest does the rest.
- **Build complex workflows with simple primitives** - [Our SDK](https://github.com/inngest/inngest-js) provides easy to learn `step` tools like [`run`](https://www.inngest.com/docs/reference/functions/step-run?ref=github-inngest-readme), [`sleep`](https://www.inngest.com/docs/reference/functions/step-sleep?ref=github-inngest-readme), [`sleepUntil`](https://www.inngest.com/docs/reference/functions/step-sleep-until?ref=github-inngest-readme), and [`waitForEvent`](https://www.inngest.com/docs/reference/functions/step-wait-for-event?ref=github-inngest-readme) that you can combine using code and patterns that you're used to create complex and robust workflows.

[Read more about our vision and why this Inngest exists](https://www.inngest.com/blog/inngest-add-super-powers-to-serverless-functions)

<br />

## Quick Start

👉 [Read the full quick start guide here](https://www.inngest.com/docs/quick-start?ref=github-inngest-readme)

1. [NPM install our SDK for your typescript project](https://github.com/inngest/inngest-js): `npm install inngest`
2. Run the Inngest dev server: `npx inngest-cli@latest dev` (This repo's CLI)
3. [Integrate Inngest with your framework in one line](https://www.inngest.com/docs/sdk/serve?ref=github-inngest-readme) via the `serve()` handler
4. [Write and run functions in your existing framework or project](https://www.inngest.com/docs/functions?ref=github-inngest-readme)

Here's an example:

```ts
import { Inngest } from "inngest";

const inngest = new Inngest({ name: "My App" });

// This function will be invoked by Inngest via HTTP any time the "app/user.signup"
// event is sent to to Inngest
export default inngest.createFunction(
  { name: "User onboarding communication" },
  { event: "app/user.signup" },
  async ({ event, step }) => {
    await step.run("Send welcome email", async () => {
      await sendEmail({
        email: event.data.email,
        template: "welcome",
      });
    });
  }
);

// Elsewhere in your code (e.g. in your sign up handler):

inngest.send({
  name: "app/user.signup",
  data: {
    email: "test@example.com",
  },
});
```

That's it - your function is set up!

<br />

## Project Architecture

Fundamentally, there are two core pieces to Inngest: _events_ and _functions_. Functions have several subcomponents for managing complex functionality (eg. steps, edges, triggers), but high level an event triggers a function, much like you schedule a job via an RPC call to a queue. Except, in Inngest, **functions are declarative**. They specify which events they react to, their schedules and delays, and the steps in their sequence.

<br />

<p align="center">
  <img src=".github/assets/architecture-0.5.0.png" alt="Open Source Architecture" width="660" />
</p>

Inngest’s architecture is made up of 6 core components:

- **Event API** receives events from clients through a simple POST request, pushing them to the **message queue**.
- **Event Stream** acts as a buffer between the **API** and the **Runner**, buffering incoming messages to ensure QoS before passing messages to be executed.<br />
- A **Runner** coordinates the execution of functions and a specific run’s **State**. When a new function execution is required, this schedules running the function’s steps from the trigger via the **executor.** Upon each step’s completion, this schedules execution of subsequent steps via iterating through the function’s **Edges.**
- **Executor** manages executing the individual steps of a function, via _drivers_ for each step’s runtime. It loads the specific code to execute via the **DataStore.** It also interfaces over the **State** store to save action data as each finishes or fails.
  - **Drivers** run the specific action code for a step, e.g. within Docker or WASM. This allows us to support a variety of runtimes.
- **State** stores data about events and given function runs, including the outputs and errors of individual actions, and what’s enqueued for the future.
- **DataStore** stores persisted system data including Functions and Actions version metadata.
- **Core API** is the main interface for writing to the DataStore. The CLI uses this to deploy new functions and manage other key resources.

And, in this CLI:

- The **DevServer** combines all the components and basic drivers for each into a single system which reads all functions from your application running on your machine, handles incoming events via the API and executes functions, all returning a readable output to the developer.

For specific information on how the DevServer works and how it compares to production [read this doc](/docs/DEVSERVER_ARCHITECTURE.md).

<br />

## Community

- [**Join our online community for support, to give us feedback, or chat with us**](https://www.inngest.com/discord).
- [Post a question or idea to our GitHub discussion board](https://github.com/orgs/inngest/discussions)
- [Read the documentation](https://www.inngest.com/docs?ref=github-inngest-readme)
- [Explore our public roadmap](http://roadmap.inngest.com/)
- [Follow us on Twitter](https://twitter.com/inngest)
- [Join our mailing list](https://www.inngest.com/mailing-list) for release notes and project updates

## Contributing

We’re excited to embrace the community! We’re happy for any and all contributions, whether they’re
feature requests, ideas, bug reports, or PRs. While we’re open source, we don’t have expectations
that people do our work for us — so any contributions are indeed very much appreciated. Feel free to
hack on anything and submit a PR.

Check out our [contributing guide](/docs/CONTRIBUTING.md) to get started.
