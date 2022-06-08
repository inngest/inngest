<p align="center">
        <br />
        <img src="https://www.inngest.com/assets/open-source/open-source-logo.svg" alt="Logo" width="220" height="90">
</p>
<p align="center">
        The event-driven queue for any language.<br />
        Created for rapid development speed and a delightful experience.
</p>

<br />

## Overview

Inngest makes it simple for you to write delayed or background jobs by triggering functions from events — decoupling your code from your queue.

At a very high level, Inngest does two things:

- Ingest events from your systems (pun *very* much intended)
- Triggers serverless functions in response to specific events — in the background, either immediately or delayed.

By building this way, you:

- Can deploy new functionality immediately, without modifying your app.
- Never have to create individual queues, workers, dispatchers, or queue-specific infrastructure again.
- Get a ton of extra features, such as event schemas, retries, historical replays, blue-green deploys, instant rollbacks, step functions, coordinated functionality etc. for free, without modifying any of your app.
- Decouple your application code from your infrastructure

We’re open source, committed to preventing vendor-lock in, using simple standards, and build in the open (we like collaboration).  Sound interesting?  

- [**Join our community for support, to give us feedback, or chat with us**](https://www.inngest.com/discord).
- [Docs](https://www.inngest.com/docs)
- [Read more about our vision and why this project exists]()

<br />
<br />

## Installing Inngest

Quick start - this downloads inngest for your os+arch into ./inngest:
```
curl -sfL https://cli.inngest.com/install.sh | sh \
  && sudo mv ./inngest /usr/local/bin/inngest
```

**Manually:** by downloading a [pre-compiled binary](https://github.com/inngest/inngest-cli/releases) and placing the binary in your path.

<br />

## Trying Inngest

Once you've installed the CLI, you can run `inngest init` to build new functions,  `inngest run` to test them, and `inngest dev` to run the dev server.

- [Quick start docs](https://www.inngest.com/docs/quick-start)

<br />

## Example usage

**Send events** to trigger background jobs anywhere in your own app:

```tsx
const host = "http://..."; // The inngest server.

// Send events via a single HTTP POST using any language.  Here's JS:
await fetch(host, {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({
    name: "user/signed.up",
    data: { id: user.id },
    user: { external_id: user.id }
  })
});
```

 

**Run a function** whenever `user/signed.up` is received (created via `inngest init`):

```tsx
import type { EventTriggers } from "./types";

export async function run({ event }: { event: EventTriggers }) {
  const { id } = event.data; // The event data sent above.

  // You can do anything here:  add the user to Stripe, Zendesk, LaunchDarkly,
  // send welcome emails, start a drip campaign - whatever you'd do
  // in the background.
  return { status: 200 };
}
```

```json
{
  "name": "Post-signup",
  "id": "abstract-nitty-a14c1d",
  "triggers": [{ "event": "user/signed.up" }]
}
```

**Deploy the function:**

```tsx
inngest deploy
```

<br />

## Architecture

Fundamentally, there are two core pieces to Inngest: _events_ and _functions_.  Functions have several sub-components for managing complex functionality (eg. steps, edges, triggers), but high level an event triggers a function, much like you schedule a job via an RPC call to a queue.  Except, in Inngest, **functions are declarative**.  They specify which events they react to, their schedules and delays, and the steps in their sequence.

<br />

<p align="center">
        <img src="https://user-images.githubusercontent.com/306177/172649986-1b3486e8-b848-4b21-bf39-2ca6faf0f933.jpeg" alt="Open Source Architecture" height="400" />
</p>

Inngest's architecture is made up of 6 core components:

- **Source API** receives events from clients through a simple POST request, pushing them to the **message queue**.
- **Message Queue** acts as an event stream between the **API** and the **Runner**, buffering incoming messages to ensure QoS before passing messages to be executed.<br />
*note: to simplify local environments this is currently absent from the DevServer, but will be included in self-hosting releases as part of the roadmap.*
- A **Runner** coordinates the execution of functions and a specific run’s **State**.  When a new function execution is required, this schedules running the function’s steps from the trigger via the **executor.**  Upon each step’s completion, this schedules execution of subsequent steps via iterating through the function’s **Edges.**
- **Executor** manages executing the individual steps of a function, via *drivers* for each step’s runtime.  It loads the specific code to execute via an **Action Loader.**  It also interfaces over the **State** store to save action data as each finishes or fails.
    - **Drivers** run the specific action code for a step, eg. within Docker or WASM.  This allows us to support a variety of runtimes.
- **State** stores data about events and given function runs, including the outputs and errors of individual actions, and what’s enqueued for the future.
- **Action Loader** loads and returns action definitions for use by the **Executor**. The source can be from disk, memory, or another persisted state.

And, in this CLI:

- The **DevServer** combines all of the components and basic drivers for each into a single system which loads all functions on disk, handles incoming events via the API and executes functions, all returning a readable output to the developer.

<br />

### Docs & Roadmap

- [You can read our docs here](https://www.inngest.com/docs)
- [Our public roadmap is part of the Inngest organization here](https://github.com/orgs/inngest/projects/1/)

<br />

### Need help?

We want to make it as easy as possible for people to write complex async functionality.  If you’re stuck, have an idea, or a feature request we’d love to hear from you.  We welcome all questions and contributions!

- **[Join our Discord community](https://www.inngest.com/discord)**, for live support and to chat with our engineers.  You can also give us real-time feedback.  It’s very much appreciated, and the best and fastest way to get involved.
    - We’ll also copy questions from here into an FAQ, and use the discussions to update our docs 😊
- [**Twitter](https://twitter.com/inngest)** and our **mailing list** for news and updates (eg. new drivers, releases, etc.)
- **Github issues**, for feedback, feature requests, and bugs.  Thank you in advance! 🙏

<br />

### Contributing

We’re excited to embrace the community!  We’re happy for any and all contributions, whether they’re feature requests, ideas, bug reports, or PRs.  While we’re open source, we don’t have expectations that people do our work for us — so any contributions are indeed very much appreciated.  Feel free to hack on anything and submit a PR.
