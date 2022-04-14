---
focus: true
heading: "Product updates:  Jan 18, 2022"
subtitle: What's fresh out of the oven recently, and what's cooking?  Here's our bi-weekly product deep dive.
date: 2022-01-19
img: "/dancing-baby-1.gif"
---

Hello fellow Inngesters!  We’re kicking off 2022 with a new process.  We’re starting bi-weekly product updates, giving you insight into the development happening over at Inngest.

## Recently released

We’re happy to announce the following features:

- **Workflow throttling.**  Even with thousands of events, specify a throttle and ensure that your workflow runs a limited number of times.  Here’s the specifics, from the product owner himself:

You can specify a throttle configuration for workflows through the `throttle_count` and `throttle_period` fields. Both fields needs a value for throttling to take effect. For example, to trigger a workflow at most once per day you can specify:

```
throttle_count: 1
throttle_period: 1d
```

Magic!  Even if you’re *flooded* with events we’ll only run your workflow once.
- **Automatic error workflows**.  Now, you can create special `error_alert` workflows.  You can have these workflows automatically run any time any of your regular workflows fail.  Have a flaky API? We’ll retry it automatically, but if it keeps failing and the workflow errors you can now specify the logic that should run when things issue out.
- **Auth with GitHub**.  Yes, we’re developers.  We (likely) have GitHub accounts.  Why not auth using GitHub?  Well, now you can.
- **Trigger expressions**.  You can now specify expressions within your trigger.  Want to only run a workflow if your order is over $500 from shopify?  Instead of handling this within the workflow, you can now specify it on the trigger itself.  We won’t even create a workflow if the data doesn’t match up.
- **UI fixes, improvements, and updates.**  We’re always working on this for you, and you’ll notice some improvements to our new dark mode.

## What’s coming

We’re also hard at work on many new features — a lot of which will improve your experience using Inngest directly.  Here’s a sneak peek:

- **Single-functon workflows.**  We’re making it effortless to react to events by letting you define single serverless functions that run as a workflow.  You can tie this into CI/CD and have a full end-to-end event-driven serverless system.  Lots to come here, including Next.JS integrations, GitHub actions integrations, etc.
- **A new event experience.**  We’re hard at work redefining how we manage your events.  It includes automatic cue type generation, json-schema generation, and SDKs from event types, plus a new browser and the ability to handle change management of events from our UI.
- **Open-source execution engine.**  We’re working on open-sourcing the engine that powers running our workflows into its own public repository.  We’ll be opening up the cue types and the executor itself — so you can run your own workflows using our CLI or your own tooling!
- **A new onboarding experience.**  We’re crafting a new experience to help you get started with event-driven serverless functions.

## Talk with us!

We’re always open to feedback.  If there’s functionality you want to see, if you have questions, or if you generally want to reach out to chat with our engineering team you can always [hop into our discord](https://discord.com/invite/EuesV2ZSnX) or [book a time with us here](https://calendly.com/inngest-thb/30min).


<div className="text-center" style={{ marginTop: 80 }}>
	<img src="/dancing-baby-1.gif" alt="" />
</div>
