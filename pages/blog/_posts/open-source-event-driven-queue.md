---
focus: true
heading: "Open sourcing Inngest"
subtitle: "The open source, serverless event-driven platform for developers."
image: "/assets/blog/open-source-event-driven-queue.jpg"
date: 2022-06-09
---

In recent years, products have grown more and more complex — which requires more engineering work to be done. Background jobs are needed in almost every system, but the dev experience is, uh, _lackluster_, to say the least. You have two typical options: a job system, or build it yourself using a message broker. Either path you choose, there’s lots of config, many services, and a lot of maintenance required to even get off the ground.

We’re bullish on our ideas on how to address the annoyances of building these systems using standard, simple interfaces and a developer-focused approach.

Today, we’re taking the the first step to open sourcing the core of Inngest so more developers can benefit from and influence it’s development. Our first release is to extract core components from Inngest’s platform and embed them in our cli as [the brand new Inngest DevServer](/blog/introducing-inngest-dev-server), which brings this better UX to your own machine. We use the same executor code internally and plan to open source additional components, like our own state store.

Let’s share some context of what those problems are that we’re taking on first, how we’re looking to improve them, and our open source plans.

## The problem we're solving

Developing message-queue or event-driven systems don’t have great developer experience. Even from a basic sense, there is pain just in selecting and planning a backing queue or event bus. They all have their tradeoffs, the different methods of configuring and different APIs or SDKs to figure out. High level, this is what you get when picking one:

**They’re commodities.** For 99.99% (_yes, 4 nines_) of applications, it’s not going to make a difference in your architecture for which system you choose. A lot of devs may think that their needs are super specific and the features of one particular system will make the difference, but at the end of the day, there won’t be a material difference if build your system on RabbitMQ, Redis, Kafka, SNS+SQS, or PubSub. It’s mostly how you choose to write your code and handle messages on top of the system. You not architecting Uber.

**You write lots of boilerplate code.** Code for publishers, consumers, polling, backoff, retry, all specific to your system and for each language.

**Configuration all the way down.** There is lots to configure starting at the system itself that you can then manage with IaC (_or not_). IAM policies for consumers/workers, dead letter queues, topics, message expiration, retention, etc. Lots of choices to make and lots of docs to read before you even get building.

**Vendor lock-in.** Each system has a different API, SDKs, message encoding format and configuration, making it less likely that you’ll ever switch, the amount of re-write probably just isn’t worth it, because, they’re commodities anyway.

**Deciding how and where to run your code.** Nothing helps you start writing code from initial development and testing to production. You’ll need to decide your own workflow and spend time automating and managing it.

These things just scrape the surface, but there’s lots to consider and you haven’t even done anything complex yet (idempotency, replay, message/event archiving, logging, etc., etc.).

## **The approach we’re taking**

We think developers deserve a better experience and a solid way to address all of this. We’re building a layer on top of common patterns in systems using standard interfaces and tools that you’re used to. It’s our goal to abstract platforms so there isn’t specific code to create lock-in. That’s what we’ve done with the Inngest platform and we’re taking the next step today.

This is what we think the system should have:

- **Simplified message or event publishing.** Just send an POST request with JSON. No queues or topics to configure up front.
- **Event/message system agnostic**. SQS, RabbitMQ, Kafka, PubSub should all be pluggable or swappable reducing lock-in.
- **Declarative, serverless consumers.** Simply declare what events/messages each consumer should handle with minimal configuration. Write your code using standard interfaces without having to run idling pools of workers.
- **Executor runtime agnostic.** Executing code should be portable and be able to be moved between Docker, AWS Lambda, Kubernetes, WASM or a combination of them.
- **Step functions support**. Breaking functionality into multiple steps with conditional logic and delays should be easy to do.

This is what you get with the existing Inngest platform today, but we want to bring this to more developers through in their local dev experience and allow developers to self-host Inngest system with swappable drivers. Today, we start our open source journey.

> If you're new to Inngest, [read about what Inngest is](/docs) or [dive into the architecture and how it works](https://github.com/inngest/inngest#project-architecture)

## Why build in the open?

We understand that most times, things built in the open benefit from more eyes and a community. Catching bugs, suggesting improvements or features, and making contributions get amplified and improve the system.

Being open sourced, we open the door for developers to be able to self-host as they desire and avoid lock-in. If you use our platform, great, if you need to hit the eject button, you can.

Open sourcing the code [allows developers to run an in-memory mini Inngest on their own machines](/blog/introducing-inngest-dev-server), making the developer process between local and production much smoother.

It also becomes open for community contributions for new drivers offering additional options for developers.

We believe in our approach and we think platform agnostic code is better. We want to bring this to more devs that we think can benefit.

## The roadmap

As we’re starting to build our a roadmap, we’re going to be focused on a few key drivers that fulfill the self-hosting goal and take it beyond an in-memory DevServer.

- PubSub driver - For the pluggable event system
- PosgreSQL driver - For executor state and history
- AWS Lambda executor runtime driver - For running the serverless consumers on AWS easily.

Our roadmap will be in the open and [you can view it right on Github here](https://github.com/orgs/inngest/projects/1).

What do you want to see next? [Let us know on Discord, we’d love to hear from you](/discord).

## Check out the repo

Head over to **[Github to check out the repo](https://github.com/inngest/inngest)**, the README, the architecture docs, and give it a spin yourself with [the quick start guide](/docs/functions).

We really hope developers get to experience what we’ve built and weigh in on the approach that we’re taking to improve working with queues and event-driven systems. We want developers to do more with less. Come chat with us!
