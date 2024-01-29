# [![Inngest](https://github.com/inngest/.github/raw/main/profile/github-readme-banner-2024-01-26.png)](https://www.inngest.com)

[![Latest release](https://img.shields.io/github/v/release/inngest/inngest?include_prereleases&sort=semver)](https://github.com/inngest/inngest/releases)
[![Test Status](https://img.shields.io/github/actions/workflow/status/inngest/inngest/go.yaml?branch=main&label=tests)](https://github.com/inngest/inngest/actions?query=branch%3Amain)
[![Discord](https://img.shields.io/discord/842170679536517141?label=discord)](https://www.inngest.com/discord)
[![Twitter Follow](https://img.shields.io/twitter/follow/inngest?style=social)](https://twitter.com/inngest)

[Inngest](https://www.inngest.com) is a developer platform that combines event streams, queues, and durable execution into a single reliability layer.

<div align="center">
  <a href="https://www.inngest.com/uses/durable-workflows?ref=org-readme">
    Durable workflows
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/ai?ref=org-readme">
    AI & LLM Chaining
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/uses/serverless-queues?ref=org-readme">
    Serverless Queues
  </a>&nbsp;&nbsp;|&nbsp;&nbsp;

  <a href="https://www.inngest.com/uses/workflow-engine?ref=org-readme">
    Workflow Engines
  </a>
</div>
<br/>

Build and ship durable functions and workflows **in your current codebase** without any additional infrastructure. Using Inngest, your entire team can ship reliable products.

- Write durable functions in your existing codebase using an [**Inngest SDK**](#sdks)
- Run the open source [**Inngest Dev Server**](#the-inngest-dev-server) for a complete local development experience, with production parity.
- The **Inngest Platform** invokes your code wherever you host it, via HTTPS. Deploy to your existing setup, and deliver products faster without managing infrastructure.

**SDKs**: [TypeScript/JavaScript](https://github.com/inngest/inngest-js) &mdash; [Python](https://github.com/inngest/inngest-py) &mdash; [Go](https://github.com/inngest/inngestgo)

---

- [Overview](#overview)
- [SDKs](#sdks)
- [Getting started](#getting-started)
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

Inngest makes it easy to develop durable functions and workflows in your existing codebase, without any new infrastructure. Inngest Functions are triggered via events &mdash; decoupling your code within your application.

1. You define your Inngest functions using the [Inngest SDK](#sdks) and serve them through a [simple API endpoint](https://www.inngest.com/docs/sdk/serve?ref=github-inngest-readme).
2. Inngest automatically invokes your functions via HTTPS whenever you send events from your application.

Inngest abstracts the complex parts of building a robust, reliable, and scalable architecture away from you, so you can focus on building applications for your users.

- **Run your code anywhere** - We call you via HTTPS so you can deploy your code to serverless, servers or the edge.
- **Zero-infrastructure required** - No queues or workers to configure or manage &mdash; just write code and Inngest does the rest.
- **Build complex workflows with simple primitives** - Our [SDKs](#sdks) provides easy to learn `step` tools like [`run`](https://www.inngest.com/docs/reference/functions/step-run?ref=github-inngest-readme), [`sleep`](https://www.inngest.com/docs/reference/functions/step-sleep?ref=github-inngest-readme), [`sleepUntil`](https://www.inngest.com/docs/reference/functions/step-sleep-until?ref=github-inngest-readme), and [`waitForEvent`](https://www.inngest.com/docs/reference/functions/step-wait-for-event?ref=github-inngest-readme) that you can combine using code and patterns that you're used to create complex and robust workflows.

[Read more about our vision and why Inngest exists](https://www.inngest.com/blog/inngest-add-super-powers-to-serverless-functions)

---

## SDKs

- **TypeScript / JavaScript** ([inngest-js](<(https://github.com/inngest/inngest-js)>)) - [Reference](https://www.inngest.com/docs/reference/typescript)
- **Python** ([inngest-py](https://github.com/inngest/inngest-py)) - [Reference](https://www.inngest.com/docs/reference/python)
- **Go** ([inngestgo](https://github.com/inngest/inngestgo)) - [Reference](https://pkg.go.dev/github.com/inngest/inngestgo)

## Getting started

👉 [**Follow the full quick start guide here**](https://www.inngest.com/docs/quick-start?ref=github-inngest-readme)

### A brief example

Here is an example of an Inngest function that sends a welcome email when a user signs up to an application. The function sleeps for 4 days and sends a second product tips email:

```ts
import { Inngest } from 'inngest';

const inngest = new Inngest({ id: 'my-app' });

// This function will be invoked by Inngest via HTTP any time
// the "app/user.signup" event is sent to to Inngest
export default inngest.createFunction(
  { id: 'user-onboarding-emails' },
  { event: 'app/user.signup' },
  async ({ event, step }) => {
    await step.run('send-welcome-email', async () => {
      await sendEmail({ email: event.data.email, template: 'welcome' });
    });

    await step.sleep('delay-follow-up-email', '7 days');

    await step.run('send-tips-email', async () => {
      await sendEmail({ email: event.data.email, template: 'product-tips' });
    });
  }
);

// Elsewhere in your code (e.g. in your sign up handler):
await inngest.send({
  name: 'app/user.signup',
  data: {
    email: 'test@example.com',
  },
});
```

Some things to highlight about the above code:

- Code within each `step.run` is automatically retried on error.
- Each `step.run` is individually executed via HTTPS ensuring errors do not result in lost work from previous steps.
- State from previous steps is memoized so code within steps is not re-executed on retries.
- Functions can `sleep` for hours, days, or months. Inngest stops execution and continues at the exactly the right time.
- Events can trigger one or more functions via [fan-out](https://www.inngest.com/docs/guides/fan-out-jobs)

Learn more about writing Inngest functions in [our documentation](https://www.inngest.com/docs).

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
