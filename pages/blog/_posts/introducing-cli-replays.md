---
focus: false
heading: "Introducing CLI Replays"
subtitle: Battle-test your local code with real production events.
image: "/assets/blog/introducing-cli-replays/header.jpg"
date: 2022-08-03
author: Jack Williams
tags: new-feature
---

Building an event-driven system can be challenging: how do you know your code will run as expected once it's deployed? With the release of [v0.5.0](/blog/release-v0-5-0?ref=blog-introducing-cli-replays), we're excited to launch `inngest run --replay`.

Replay allows you to battle-test local code against your real production data to avoid breaking changes and gives you confidence when deploying minor fixes or huge refactors.

![An example of CLI replays running](/assets/blog/introducing-cli-replays/top-example.gif)

## Zero-effort testing

The ability to replay events is an often-touted feature of many queueing systems, but it's never easy to achieve without a tonne of hand-rolled code. Enter Inngest Replays.

```bash
$ inngest run --replay
```

With a simple command, we'll build your code, pull real recent events from your Inngest Cloud account, and test your function against them. You can instantly prove that your change will work in production before you even commit your code.

We've found it particularly useful when bootstrapping new functions triggered by existing events, as you have an instant feedback loop while developing. Write code once, deploy once, done!

## Protect against breaking changes

You can also use replay in your CI, ensuring that every single deployment is validated against real, recent production data *before* it's shipped.

Events and their shape can evolve over time, be it from external services or internal teams. Replay in CI gives you an easy method to protect against accidental breaking changes regardless of the source of the data.

## Reproduce production issues locally

One of the joys of Inngest's event-driven platform is that you can isolate problem events to quickly investigate an issue alongside the data that caused it.

Replays allow you to target a single problem event and test your function against it to debug and resolve production issues locally.

```bash
$ inngest run --replay --event-id 01G8BG4FT7CZVAD38D4RJNGTT1
```

Addressing a production issue is usually an ill-choreographed dance of crawling through logs, assuming circumstances, and piecing together traces and data across your infrastructure. Or we could skip straight to solving the problem. 🤷

## Get started

Replays give you some awesome tooling for quickly and safely deploying your product. We'd love to hear how you use them on [GitHub](https://github.com/inngest/inngest) or [Discord](/discord).

Go check out the [`inngest run` docs](/docs/cli/run?ref=blog-introducing-cli-replays), or [start building today](/sign-up?ref=blog-introducing-cli-replays).
