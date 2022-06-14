---
focus: false
heading: "Building an event-driven queue based on common standards"
subtitle: The design decisions and architecture for a next-gen queuing platform
image: "/assets/blog/building-event-driven-queue.jpg"
date: 2022-06-15
---

At Inngest, we’ve built a next-gen queueing platform for modern development  — it’s event-driven, and (if you want) entirely serverless.  We’d like to talk a little bit about *why* we built this, the benefits, and the process & design decisions of building the system.

Before we begin, some background: we started out by building this platform for ourselves and we’ve open sourced it here: [https://github.com/inngest/inngest-cli](https://github.com/inngest/inngest-cli).  We’re adding to this heavily in the near future with lots more to come — so feel free to check it out!

### Current queueing problems

First up, you might wonder: why built a new queue when there’s so much out there already.  The TL;DR is that — even though a bunch of stuff exists and is battle tested — we found queues *really* annoying to manage.  The dev UX kind of sucks.

The not-so-long version is that queues can be slow to build out, full of boilerplate, annoyingly stateful, and it’s hard to build complex logic (like, step functions with conditionals or human-in-the-loop transitions).  It’s also annoying to handle multi-language support… you’ll have many ways of enqueueing work.  And expensive to pay for wasted compute.  We could rant for a while, but in essence they’re the antithesis of modern developer UX.

You might say that we wanted the experience of Vercel for our core queueing infrastructure.  

### The ideal system

And, taking a step back, we knew there were some key improvements we could make to improve the dev UX:

- Functions (workers) should be declarative, specifying how and when they should run.
- Functions should be DAGs, with each step sharing context.  When you’re building a complex app your logic also becomes more complex.  This should be built in, and easy to use.
- Functions should be serverless.  Or ran via webassembly.  Or server…full?  If you’d like, you should be able to have the flexibility to just ‘push’ your function live, without having to manage a worker pool.  Or queues.  Or scheduling work.  Death to the not-so-autoscalers, please!
- It should be easy to canary or blue-green deploy function changes (without any infra, architecture, or app changes).
- Functions should also be as easy as possible to create.  Ideally with no SDKs, no frameworks, no replacing standard libraries, and so on.  Just, like, write your code and go, please!
- The data passed into functions should be strongly typed.  And should automatically alert you when it’s incorrect, without you having to write your own SDK.  And… data should be versioned, so you can handle changes over time.
- It should be *easily* locally testable (eg. from zero to set up in < 10s).  And testable in CI.  No compromises here.
- Also, an obscure one learned via experience:  if possible, function runs should map directly to the responsible user.  It’s a huge boon to customer support & debugging if you can see invocations for each particular customer.

There’s also the core requirements of queuing systems in general:  availability, handling failures, scale, etc.

### Designing the system

So, how do we go about creating a new platform which achieves all of this (and more)?

In not so many words, indirection.  Joe Armstrong had a *really* good Erlang talk on “[the how and why of fitting things together](https://youtu.be/ed7A7r6DBsM?t=307s)” on abstraction in systems which describes the *contract* between systems.  In our case, the primary contract between dispatching work on a queue and performing the work is *a strongly typed event*.  Let’s dive in:

- Instead of treating queues as ‘eventual RPCs’, we embrace event-driven design.  This decouples the callsite, the specific queue, and the function.  If you want to invoke a job, send an event.  If you want to delay some work, send an event.  If you want to handle an incoming webhook, send an event.  It’s all easy.
- By using events, we can enforce schemas for events, and we can version these schemas.  This already solves several of our requirements.  We can take this a step further and do Segment-like *persona identification*, giving us “personalized debugging” on a user-by-user level.
- Functions can specify which events trigger their own execution, with optional conditional expressions.  This makes functions declarative.  It also means many functions can be invoked by one trigger, which means we can deploy new asynchronous functionality without changing any of the core services.
- If functions specify their triggers, the execution engine in the queue itself must be responsible for handling incoming events and doing the enqueueing.  By moving the scheduling into the execution engine of the queue, we can add function version control, blue-green deploys, canary deploys, rollback, etc. to our platform itself.
- Because we’re event-driven, we can store the event payloads which trigger functions for an arbitrary period of time.  This lets you do things like historical replay — in which you can use production (or prod-like) data in your local dev environment to ensure your changes work as expected.
- Finally, because we’re event-driven we can implement *really complex user flows*.  Things like:
When this specific event arrives, run ‘step A’.  Wait for another specific event from the same user, then run ‘step B’.  And if we don’t receive this other event, run ‘step C’.  We call this ‘event coordination’ because we’re literal and naming this feature is hard.  Essentially, this lets you pause workflows and do all kinds of coordination.

In general, by making a simple change — moving from RPC to event-driven — we’re able to completely change how the system works from a developer point of view, and we’re able to unlock previously complex functionality with minimal work.

Now, there is _some_ prior art out there for event-driven queues.  You can tie in SQS with Lambda, or Pubsub to Cloud Functions.  That doesn't achieve the rest of our objectives, though - and you're still left with building out the SDKs and logic to bundle functions together in many steps, or do rollouts. 

### Striving for standards & simplicity

Finally, in continuing with making the developer experience as good as possible, we want our system to be easy to learn.  We’re believers in the rise of the zero-SDK – no SDK needed.

Here’s what we don't want you to have to do:  set up a bunch of protobuf in every one of your repos.  Or debate which AMQP library you should use.  Or learn a new SDK which replaces your language’s internals.

The easiest option for sending events in this case is a single HTTP2 POST request using JSON.  It’s built in to every language, requires no tools to learn, and you only need to add 2 things:  your host URL and your API key.  The rest is handled.  (At scale you might want to switch out to using Kafka and such, which is very much okay too, because the underlying broker is also fungible).

And what about functions?  We also added indirection here, too.  Each function has one or more steps;  they’re DAGs.  Here’s how you define a function:

```json
{
  "name": "A beautiful background function",
  "id": "prompt-deer-ede40d",
  "triggers": [
    {
      "event": "user/signed.up",
      "expression": "user.created_at > '2020-01-01'"
    }
  ],

  "steps": {
    "step-1": {
      "id": "step-1",
      "path": "file://./steps/send-sms-dispatch",
      "name": "Send SMS",
      "runtime": {
        "type": "docker"
      }
    },
    "step-2": {
      "id": "step-2",
      "path": "file://./steps/add-to-intercom",
      "name": "Add user to Intercom",
      "runtime": {
        "type": "lambda"
      },
      "after": [{ "step": "step-1" }]
    }
  }
}
```

As you can see, this (contrived) function has two steps in a waterfall-style sequence.  Further, each step in a function has its own runtime.  We have multiple runtimes that specify how to execute functions.

If you have functions which you want to host on Lambda, that works.  If you have containers, that’s great too!  [To keep things simple we pass in the event & function context as an argument.  We’ll read the step’s output from stdout](https://www.inngest.com/docs/functions/function-input-and-output).  Again, no SDK required — it’s all language builtins.

### Wrapping up

We’ve been using this for a while and it’s been much easier to develop and deploy our asynchronous tasks.  

While it seems like a simple change there’s a lot that’s needed to make the system work.

We’ve open sourced the execution engine, locally running drivers, in-memory and Redis-backed state, and we’re also open-sourcing an abstraction over many common queueing layers (eg. Celery, Faktory) to make gradual adoption easier.  We’re also open-sourcing new runtime drivers such as an AWS Lambda layer, and we have plans for webassembly adoption in the future.

You can check out the code here: [https://github.com/inngest/inngest-cli](https://github.com/inngest/inngest-cli).  We’ll be adding *lots* to this over a short amount of time.

We’re *super* interested in feedback — your ideas, thoughts, and comments are more than welcome, either in GitHub or [our discord.](https://www.inngest.com/discord)

If you’d like to self host this and prototype it, drop us a note in Discord and we’ll help you get set up.  We also offer a [hosted version](https://www.inngest.com), but this isn’t a sales pitch: we’re genuinely interested in this technology and sharing our approach because we think it offers a host of benefits (though we’re biased).
