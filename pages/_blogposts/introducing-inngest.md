{
  "slug": "introducing-inngest",
  "redirects": [],
  "title": "",
  "popular": true,
  "order": 1,
  "date": "October 2021"
}

~~~

# Introducing Inngest:  an event workflow platform

<div className="blog--callout">
We’re launching Inngest, a platform designed to make building event-driven systems fast and easy.

First, what is Inngest?  Inngest is a serverless event platform.  It aggregates events from your internal and external systems, then allows you automatically run serverless code in response to these events - as a multi-step workflow.  

It’s like putting GitHub Actions, Lambda, Segment, and Zapier in a blender.  You can sign up to Inngest for free and start today by [visiting here](https://app.inngest.com/register).
</div>

## Why events?

Events drive the world.  They describe exactly what happens in all of your systems.  When a user signs up, that’s an event.  When a user pays - or fails to pay - that’s an event.  When you update a task, or a Salesforce lead - that’s an event.

Events represent everything, as it happens in real time.  For the engineers, it’s also decoupled from the implementation - you can change how signup works, but the event is still the same (someone has still signed up, after all).

Unifying and working with these events makes a lot possible.

## What can it be used for?  Some examples…

High level, when you begin working with events you can react to anything - in real time.  That means you can do things like:

* Build real-time sync between platforms as things update
* Run workflows when users sign up, handling the emails, marketing, billing organization, and internal flows asynchronously
* Process leads, enrich data, and assign tasks when your company gets new leads

And when you unify your events, you can coordinate between multiple events (or their absence):

* When a user signs up but you don’t receive a sign-in event within 7 days, run a churn campaign and send emails to the user.
* When a shipping label is generated but no shipment sent event is received by the label expiry, generate a new label and email the user.

## Some extra benefits

By building with events as a first class citizen you also get:

* Reproducibility
* Traceability
* Easy debugging
* Retries
* Audit trails
* Versioning and change management
* Event coordination

The plumbing exists for this in common platforms.  You can use SQS, Kafka, Lambda, Terraform. etc. to start building event-driven systems yourselves.  Though, it can take some time to build the debuggability, audit trails, and versioning you get out of the box with Inngest.

We’ll dive into more of these pieces in future posts.  If you’re interested in getting started with a serverless event platform in minutes, check out https://www.inngest.com.
