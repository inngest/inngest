---
category: "Events"
title: "via Webhooks"
slug: "event-webhooks"
order: 4
hide: true
---

<div className="tldr">

**Webhooks allow you to receive events from external systems via an HTTP call immediately, with zero infrastructure.** You can transform incoming events on-the-fly using javascript (ES6+) to match our event format.

We provide set of built-in webhooks for common services. Additionally, we offer integrations for some services which receive events automatically.

</div>

Webhooks allow you to create a new URL for receiving events from other systems, with zero infrastructure, servers, or code.

## Why use Inngest for webhooks?

Using Inngest to manage incoming webhooks is easier, faster, and more reliable than building out webhook infrastructure yourself.

**Simplicity**: You can create a new webhook within seconds. Each event sent to the webhook is tracked within Inngest and can trigger workflows immediately.

**Speed**: When you create a new webhook in Inngest, we can immediately start receiving events and running workflows.

**Reliability**: We process incoming webhooks via our ingest API, which maintains high availability to receive every event you are sent. We handle workflow retries and event replays for you on our side.

## Creating webhooks

You can create a new webhook within Inngest by heading to "Sources" and [selecting "Webhooks" tab](https://app.inngest.com/sources/new#Webhook).

When creating a webhook you will be prompted for it's name and can optionally specify a transform and filter list. A transform allows you to change the event's structure before we process it, and filters allow you to allow or deny specific events or IPs from using the webhook.

## Transforms

Transforms allow you to change an incoming event's structure before we process it. Each event we process must match our [event format](/docs/event-format-and-structure) by having **at least** the `name` and `data` field. You can specify ES6+ [JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript) code which transforms an incoming event when creating a webhook.

The transform **must return an object containing a `name` and `data` field**:

```javascript
function transform(evt, headers = {}) {
  const name =
    headers["X-Github-Event"] || evt?.headers["X-Github-Event"] || "";

  return {
    // Use the event as the data without modification
    data: evt,
    // Add an event name, prefixed with "github." based off of the X-Github-Event data
    name: "github." + name.trim().replace("Event", "").toLowerCase(),
  };
}
```

<div>
	<img src="/assets/webhook-transform.png" alt="Webhooks in Inngest" />
	<small>An example of a webhook transform, adding the event name based off of the incoming webhook data</small>
</div>

<br />

The above example shows a GitHub transform, using the incoming event data to create a new `name` field.

You can use almost all ES6 features, and our UI allows you to preview the transform prior to saving the webhook. Transforms can be modified at any time.
