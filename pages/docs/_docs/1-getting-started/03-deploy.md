---
category: "Getting started"
title: "Deploying functions"
slug: "deploying-fuctions"
order: 10
---

Once you've built and locally tested your function it's time to deploy it to the cloud.

Deploying your function will automatically build each step, push the code to Inngest,
and create a new version of the function.  Inngest will automatically run this new
version every time matching events are received.

Read on to learn about:

- Test and production environments
- How to deploy functions

## Environments

Inngest comes with both a test and production environment out of the box.  This allows
you to deploy functions safely without affecting your customers, then promote and deploy
to production when it's ready.

## Running deploy

You can deploy functions by running <b>`inngest deploy`</b>.  By default this deploys
to the test environment.  You can deploy to production via:

```sh
# deploy all functions in the local directory, recursively, to prod.
inngest deploy --prod .
```

This will package and ship all functions to production, setting them live instantly.
