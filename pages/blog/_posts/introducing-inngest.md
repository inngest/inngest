---
# focus sets this blog post as the blog focus.  The latest post will be focused if there's
# > 1 focus post.
focus: true
heading: "Introducing Inngest:  an event workflow platform"
subtitle: We’re launching Inngest, a platform designed to make building event-driven systems fast and easy.
date: 2021-10-05
order: 1
---


<div className="blog--callout">

We’re launching Inngest, a platform designed to make building event-driven systems fast and easy.

First, what is Inngest?  Inngest is a serverless event platform.  It aggregates events from your internal and external systems, then allows you automatically run serverless code in response to these events - as a multi-step workflow.  

It’s like putting GitHub Actions, Lambda, Segment, and Zapier in a blender.  You can build server-side functions and glue code in minutes, with no servers.  If you're interested, you can sign up to Inngest for free and start today by [visiting here](https://app.inngest.com/register).

</div>

## Why events?

Events are powerful: they describe exactly what happens in every system.  When a user signs up, that’s an event.  When a user pays (or... fails to pay), that’s an event.  When you update a task, or a Salesforce lead, or a GitHub PR, that’s an event.

So, **events represent things as they happen in real time**.  You're probably familiar with them already because, well, product analytics has been a thing for some time.

But they're powerful not just because of analytics.  They're powerful because **your systems often need to run a bunch of logic when things happen**.  For example, when a user signs up to your account you might need to add them to your billing system, add them to marketing lists, add them to your sales tools, add them to your CRM, and send them an email.

To start, you might chuck the first integration in your API controller.  Or a goroutine.  And as things progress you might start building out queues for each integration.  Or, if you're all in on microservices, you might want to build a lot of infrastructure around sending messages - events.

The beautiful thing about events is that **events are decoupled from the actual implementation that creates them**.  You can change how signup works (oauth, magic links, or - have mercy - saml), but the event is still the same.  Events grant you freedom.

Unifying and working with these events makes a lot possible.

## What is Inngest?

So events are great.  They let you know what's happening.  They provide audit trails when things happen.  **But event-driven systems can be difficult to build**.  And they're very difficult to audit and debug.  Don't get us wrong:  if you want to wrangle with Terraform, maybe set up Kafka (I have a soft spot for NATS), build your publishers, subscribers, service discovery, throttling, retries, backoff, and other stuff, it can be done. But it's not exactly "move fast", even if it is very much "break things".  You also don't get webhook handling, integrating with external services, change management, or non-technical insight here for free either.

Well, this is where we step in.  **Inngest provides you with a serverless event-driven platform and DAG-based serverless functions out of the box.**  Send us events - any and all of them from your own systems.  Connect webhooks up to external systems.  And then build out serverless functions that run in real-time whenever events are received.  That's the short version.

The long version is that you can:

- Coordinate between events easily (eg. "wait up to 1h for a customer to respond"), while also handling timeouts
- Version each workflow, and roll back instantly if need be
- Visually see and understand the workflows
- Get retries and error handling out of the box
- Step-debug thtourhg your workflows
- Run _any_ code as part of a workflow, in any language
- Automatically create schemas for each of your events
- Collaborate and hand-off workflows to non-developer folk

We’ll dive into more of these pieces in future posts.  If you’re interested in getting started, you can [sign up here](https://app.inngest.com/register).

## What can it be used for?  Some examples…

Let's make things concrete.  There are a few examples that are extremely common:

- On signup, propagate the new account to external systems (sales, marketing, customer support, billing) and send emails
- On signup, wait for X events to happen or begin a churn workflow (drip campaigns, etc).
- On payment, send receipts and update external systems
- When new customer support requests are received, run escalation logic procedures based off of the user's account, run NLP to detect importance, tone, and category of request

There are also a bunch of things you might need to do depending on your sector:

* When a return label is generated but no shipment sent event is received by the label expiry, generate a new label and email the user.
* When a meeting is upcoming, send reminders

It goes on, and on, and on, depending on what you're doing.  And we're here to help you build it.  You can [get started for free](https://app.inngest.com/register), or if you're interested in chatting with us you can send us an email: <a href="mailto:founders@inngest.com">founders@inngest.com</a>
