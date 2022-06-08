---
hide: true
focus: true
heading: "Introducing Inngest DevServer"
subtitle: "The first tool purposely designed for event-driven asynchronous system local development"
image: "/assets/blog/introducing-inngest-dev-server/featured-image.jpg"
date: 2022-06-09
---

At Inngest, we think that writing async app logic should be easier than it actually is.

No matter what background jobs you're writing, if you want it to be reliable you're going to have to pick a (often language specific) queueing system, build out the infrastructure locally, in test, and in prod, write your queues, subscribers, build out workers, and so on. It's quite laborious, to say the least, and once you're done you're very much tied to your chosen implementation.

We've been hard at work building a new platform to make this easier — _an event-driven queue for any language_. We make it simple for you to write delayed or background jobs by triggering serverless functions from events — decoupling your application code from your queue.

And today, we're releasing a 1-1 copy of our executor and runtime for local use, called the Inngest DevServer. We're also building the DevServer in the open — [you can read more here](/blog/open-source-event-driven-queue).

What does that mean? Not to bury the lede: it means you can run one command locally (`inngest dev`) to run a local, in-memory copy of Inngest and never have to worry about setting up queuing infrastructure or workers again.

In this post, we'll walk through the DevServer in more detail and show you it's basic usage.

## What is the DevServer?

The DevServer is the full stack Inngest architecture running on your machine that does four key things:

1. It runs a dead simple API for sending (_publishing_) events
2. It loads all of your asynchronous code as functions and step functions
3. As events are received, your functions are executed with the event payload
4. You see all events and function's results immediately in the DevServer output

There is no need to run a queue or an event stream and configure publishers or consumers - just send events with an HTTP `POST` request and it will run your code. Interested? Let's go a step deeper and explain how it works.

## How does it work?

![Inngest DevServer architecture diagram](/assets/blog/introducing-inngest-dev-server/open-source-architecture.jpg)

Under the hood the DevServer does a few things to make it all come together.

First, the DevServer recursively searches your project directory for functions configured with `inngest.json` (or `.cue`) files ([docs](/docs/functions/configuration)). As each function is found, the DevServer does two things:

- Each function's event trigger (or scheduled trigger) is registered by the DevServer. This creates a map of events-to-functions.
- Functions are built using each's own `Dockerfile`(s) just as with the `inngest run` command ([docs](/docs/cli/run)).

Simultaneously, an API server is started on `localhost:9999` (or another port of your choosing). This API matches the production Inngest Source API for ingesting event payloads.

As your application (e.g. a Next.js API or Flask backend) sends an event to the API server, the DevServer finds matching functions within its map of event triggers. Every matching function is then executed in a container, passing the event payload as JSON via `stdin`. Whatever is written to `stdout` or `stderr` is captured as the function's response.

The function's response as well as the event payload itself are all written to the DevServer's log output so you can easily see the inputs and outputs of your background tasks.

We think that this is a simple, yet powerful, workflow for developers and [the code is fully open source on Github](https://github.com/inngest/inngest-cli) for you to dig even deeper if you want. We've also designed the DevServer so we can add new non-Docker runtimes (_think Lambda_) and we have plans to add many more features in the coming weeks and months ([check out our roadmap here](https://github.com/orgs/inngest/projects/1)).

## How can I get started?

1. Download the Inngest CLI:

```
curl -sfL https://cli.inngest.com/install.sh | sh && sudo mv ./inngest /usr/local/bin/inngest
```

2. Navigate to your project repo and create a new function and enter a custom name for your event:

```
cd my-project
inngest init --event my.event.name
```

3. Start up the dev server:

```
inngest dev
```

4. Send an event to the dev server with any placeholder key (http://localhost:9999/e/KEY). In this example we'll use JavaScript's fetch:

```js
await fetch("http://localhost:9999/e/KEY", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    // replace this with your event name and the data you want to send
    name: "my.event.name",
    data: { hello: "there" },
    user: { email: "test@example.com" },
  }),
});
```

_Note - You can also use [one of our language SDKs](https://www.inngest.com/docs/sending-data-via-inngest-sdks) if you prefer_

You should now see the event in the DevServer's output and your new function should have run!

![Inngest DevServer example output](/assets/blog/introducing-inngest-dev-server/inngest-dev-server-output-example.png)

## OK - Just show me the code

You probably just want to see some real code don't you? We created a demo project with a Next.js backend and a function that sends an SMS via Twilio's API. Check it out here: [github.com/inngest/demo-nextjs-full-stack](https://github.com/inngest/demo-nextjs-full-stack).

You can also view the DevServer's source code from right in our CLI repo: [github.com/inngest/inngest-cli](https://github.com/inngest/inngest-cli).

**Have questions, feedback or ideas? [Join our Discord](https://www.inngest.com/discord)!**

## Over to you

That's it - we think the Inngest DevServer is the one tool that you can use to build and test your asynchronous code from end-to-end with zero configuration and setup. This is just the beginning and we're excited to bring more features and more power to developers hands in the months ahead!

We have plenty more in store and would love to hear from you to shape the future of Inngest and the DevServer - come say hi in our Discord:
