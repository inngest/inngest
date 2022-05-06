---
heading: "Programmable event platforms"
subtitle: Programmable event platforms allow you to build serverless event-driven systems in minutes.  Here's an introduction to them.
date: 2022-01-10
image: "/img/globe.png"
---

Software architecture has undergone rapid changes in the last decade. “Table stakes” for products has advanced and with it our software has become increasingly complex. To make this work, we’re integrating more tools than ever, and we’re building more complex architectures encompassing (micro)services, serverless, and event-driven systems.

The advent of serverless, Kafka, and event-driven programming has been incredibly helpful in helping us succeed as engineers. That said, at Inngest we’re not (quite) happy with the current state of event-driven platforms and serverless. It's easy for events to propagate with no schemas or change management, or for serverless to turn into a complex rube goldberg machine, making changes and debugging next-to impossible. Features that were previously easy become service hell, with development split over queues, messaging, subscribers and workers.

Even with the underlying platforms and technology advancing (thanks Kafka, Pulsar, CF Workers, etc!) there’s still a gap in how we fundamentally build software for our users. At Inngest, we’re not happy at the developer experience for these systems, and we feel that developers deserve better.

## Introducing Inngest: a programmable event platform

We’re thrilled to preview our new serverless programmable event platform, making building serverless event-driven systems easy. Inngest subscribes to all of your events and runs serverless functions any time specific events are received. We let you focus on writing your business logic without worrying about building or managing event-driven infrastructure.

How does Inngest work? We provide of the queues, subscribers, workers, backoffs, retry logic, schema management, event replays, and audit trails out of the box. We let you see which events trigger which worfklows, when workflows were live, and which users trigger which workflows. We also let you write and deploy your serverless functions in any language, which we'll run for you.

It’s not just lambda: we allow you to run a complex [DAG](https://en.wikipedia.org/wiki/Directed_acyclic_graph) of serverless functions, every time events are received. Events can be anything — internal API calls, subscriptions to your current infrastructure, custom webhooks, OAuth service integrations, or (if you're into it) web3 events.

Your DAG is defined via a strictly-typed config (_not_ YAML), so we can validate and verify your config statically (and locally). It handles coordination between independent events (wait for this for some time) and can run custom code in any language as part of your workflow. Here’s a summary of the functionality we’ve built for our preview:

- Event coordination, so you can create complex flows that rely on multiple events within a specific time period (and handle timeouts)
- Audit trails, by automatically logging the users that are responsible for each event
- Auto-generated event schemas, which evolve as event versions change
- Automatic retries, with custom error handling logic when things continue to fail
- Event replay, by storing each event received for up to 6 months
- A step-over debugger for running each part of your workflow
- Version control built-in, with one-click rollbacks and the ability to schedule deploys in the future
- An advanced UI for visualization, debugging, and handoff to other technical teams
- A library of existing workflows for common functionality
- Pre-built integrations for faster buildout and iteration

Critically, we’re **not** replacing your current infrastructure. Our goal is to empower you and your team to build maintainable complex software — faster than ever, and without compromise. You can start sending events through us for free, then deploy workflows whenever you're ready. It's additive, and aimed to make you build faster.

Give us a try by signing up today. We’re free during the preview: all we ask is for your thoughts and feedback to make it better. And, for accounts that sign up during preview, we’ll grandfather you a 25% discount on any plans in the future.
