# Inngest

![Latest release](https://img.shields.io/github/v/release/inngest/inngest?include_prereleases&sort=semver)
![Test Status](https://img.shields.io/github/actions/workflow/status/inngest/inngest/go.yaml?branch=main&label=tests)
![Discord](https://img.shields.io/discord/842170679536517141?label=discord)
![Twitter Follow](https://img.shields.io/twitter/follow/inngest?style=social)

Run reliable serverless functions in the background. Inngest allows you to schedule, enqueue, and run serverless functions in the background reliably with retries on any platform, with zero infrastructure, fully locally testable. Learn more: https://www.inngest.com.

<br />

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Project Architecture](#project-architecture)
- [Community](#community)

<br />

The local development UI:

![DevUI](https://user-images.githubusercontent.com/306177/204876780-d97eec85-53e2-4fca-81ce-cae45d56c319.png)

<br />

## Overview

Inngest makes it simple for you to write delayed or background jobs by triggering functions from events — decoupling your code within your application.

- You send events from your application via our SDK (or with a Webhook)
- Inngest runs your serverless functions that are configured to be triggered by those events, either immediately, or delayed.

Inngest abstracts the complex parts of building a robust, reliable, and scalable architecture away from you, so you can focus on writing amazing code and building applications for your users.

We created Inngest to bring the benefits of event-driven systems to all developers, without having to write any code themselves. We believe that:

- Event-driven systems should be _easy_ to build and adopt
- Event-driven systems are better than regular, procedural systems and queues
- Developer experience matters
- Serverless scheduling enables scalable, reliable systems that are both cheaper and better for compliance

[Read more about our vision and why this project exists](https://www.inngest.com/blog/open-source-event-driven-queue)

<br />

## Quick Start

👉 [Read the full quick start guide here](https://www.inngest.com/docs/quick-start?ref=github-inngest-readme)

1. [NPM install our SDK for your typescript project](https://github.com/inngest/inngest-js): `npm install inngest`
2. Run the Inngest dev server: `npx inngest@latest dev` (This repo's CLI)
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

- The **DevServer** combines all the components and basic drivers for each into a single system which loads all functions on disk, handles incoming events via the API and executes functions, all returning a readable output to the developer. (_Note - the DevServer does not run a Core API as functions are loaded directly from disk_)

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
