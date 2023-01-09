---
heading: "Inngest: OS v0.5 released"
subtitle: This release contains exciting new functionality, including replay and our self-hosting services
date: 2022-07-26
image: "/assets/blog/release-v0.5.0.jpg"
tags: release-notes
---

[Inngest v0.5.0 is here](https://www.github.com/inngest/inngest)! This release contains _exciting_ new functionality to improve your lives as a developer, as well as routine improvements. Some of the highlights which we’ll dive into:

- **Historic replay,** which allows you to locally test your functions with _real production data_
- **Self-hosting beta,** so that you can host Inngest in your own environment

Read more about our [future plans in our roadmap](https://github.com/orgs/inngest/projects/1), and if you want to propose new features or ideas feel free to [start a discussion](https://github.com/inngest/inngest/discussions) or [chat with us in discord](/discord). Let’s dive in!

## Replay past events

This release brings an exciting new feature to `inngest run`: **easily testing your local functions against real, production data.**

This lets you ensure that your function works exactly as intended with real events that are flowing through your system — giving you more confidence than relying on unit testing or dummy data only.

Best of all, it’s really simple to use:

```
inngest run --replay
```

How is this possible? Inngest is event-driven, and we store all of the events that flow through your system. This lets us take those historic events and pipe them through to your local functions. It’s a completely different approach than you might be used to with eg. SQS or RabbitMQ, which enables much better development practices than previously available.

You can ~~read the documentation for historic replay here~~ (**NOTE** - This has been deprecated in favor of [the Inngest SDK](/docs/quick-start)).

## Self hosting beta

While we offer our [hosted cloud](/sign-up?ref=v0.5.0) which lets you start using Inngest in minutes, we’ve also added a new command to the CLI: `inngest serve`. This lets you run the core Inngest services to accept events, initialize functions, execute functions, and deploy new versions to your own infrastructure. The backends are entirely configurable; you can choose any messaging system for processing incoming events by [changing your config file](https://github.com/inngest/inngest/blob/main/pkg/cuedefs/config/config.cue).

[We’ve included example self-hosting stacks](https://github.com/inngest/inngest/tree/main/hosting-stacks/), which include all of the terraform and configuration you need to get started. We’ve also added some benchmarking:

- A single 1GB / 0.5vCPU event API can process 110 requests per second with a p99 latency of 35ms, without breaking ~35mb ram usage.
- It’s easy to scale to thousands of requests per second, as the services themselves are shared nothing.

If you’re interested in self-hosting, you can [read the docs here](/docs/self-hosting) and [chat with us on discord](/discord) if you have any questions

## Other changes

We’ve also made several changes to the open-source state interface. We now include a distributed waitgroup which tracks the number of outstanding steps in a function. This lets the `inngest run` command know when a function is complete — necessary for a smoother dev UX.

We’ve also changed the way the dev server works under the hood. It now better matches self hosting environments by using the exact same services as in self-hosting.

Happy building!
