---
category: "Managing workflows"
title: "Deploying serverless functions"
slug: "deploying-serverless-functions"
order: 3
hide: true
---

This tutorial will take you through deploying your own serverless functions from scratch.

We're going to be deploying a basic typescript function that gets the weather from workflow metadata, though any language can run on Inngest. Here's the code:

```typescript
async function main() {
  // Get the city from workflow metadata.  This is configured in the side panel.
  const args = JSON.parse(Deno.args[0] || "{}");
  const city = args?.metadata?.city;

  if (!city) {
    console.log(JSON.stringify({ error: "no city present" }));
    // Exit with exit code 0, which is the success exit code.  This prevents
    // the action from being retried; the workflow was incorrectly configured
    // and retrying will not fix.
    return;
  }

  let response = {};

  try {
    const result = await fetch(`https://wttr.in/${city}?format=4`);
    const text = await result.text();
    response = { weather: text };
  } catch (e) {
    console.log(JSON.stringify({ error: e.toString() }));
    // Exiting with a non-zero exit code will retry the action up to 3 times
    // by default, with exponential backoff.
    Deno.exit(1);
  }

  // console.log writes to stdout, which is how the output is captured.
  // We also expect a single JSON object as the output.
  console.log(JSON.stringify(response));
}

main();
```

Of course, your action can do anything you need: resize files, run audio-to-text machine learning models, enriching data - whatever you want to code.

## Prerequisites

1. Install inngestctl: ` curl -sfL https://raw.githubusercontent.com/inngest/inngestctl/main/install.sh | sh`
2. Log in to your inngest account via inngestctl: `inngestctl login -u your@email.address`

## Deploying to Inngest

There are only three steps to deploy your serverless function on Inngest, provided it
[is implemented correctly](/docs/actions/serverless/intro#implementation):

1. Create a Dockerfile and containerize your code
2. Create a new Inngest action configuration file (`inngestctl actions new $name > action.cue`)
3. Deploy via inngestctl (`inngestctl actions deploy ./action.cue`)

Deploying is as quick as pushing the Docker image, then it's live!

Let's walk through the steps:

### Create a Dockerfile

For our example we're running Deno, so here's a Dockerfile which runs our app:

```docker
FROM denoland/deno:1.12.2

# Add our index.ts file containing the main function
ADD ./index.ts /index.ts

# Run the index file, allowing network access - and prevent Deno from
# adding its own output via "--quiet"
ENTRYPOINT ["deno", "run", "--allow-net", "--quiet", "index.ts"]
```

If we build this docker image as **inngest-serverless** and run it you'll se some output:

```
$> docker build -t inngest-serverless .
$> docker run -ti --rm inngest-serverless '{"metadata":{"city":"London"}}'
{"weather":"London: ☀️ 🌡️+61°F 🌬️↗8mph\n"}
```

### Create an action configuration file

In order to know which container you want to run and how it is configured within a workflow, we must add some [configuration](/docs/actions/serverless/intro#configuration).

You can generate an empty config file by running:

```
inngestctl actions new $name > action.cue
```

This will save the empty configuration to action.cue. Remove ` > action.cue` to show the output without saving it.

The action configuration looks like this:

```json
package main

import (
        "inngest.com/actions"
)

action: actions.#Action
action: {
  dsn:  ""
  name: ""
  version: {
    major: 1
    minor: 1
  }
  workflowMetadata: {}
  response: {}
  edges: {}
  runtime: {
    image: "your-docker-image"
    type:  "docker"
  }
}
```

Let's fill this in:

```json
package main

import (
        "inngest.com/actions"
)

action: actions.#Action
action: {
  // An action has two parts: your account identifier, a slash, and then
  // a unique action name.  Fill this with your own account info.
  dsn:  "your-account-identifier/your-action-name"
  name: "Get the weather"
  version: {
    major: 1
    minor: 1
  }

  workflowMetadata: {
    city: {
      type: "string"
      required: true
      // form defines the workflow side panel UI elements to show
      form: {
        title: "What city should we find the weather for?"
        placeholder: "New York"
      }
    }
  }

  response: {
    // The JSON response from our action has a weather key and an error key.
    weather: { type: "string" }
    error: { type: "string" }
  }

  edges: {}

  runtime: {
    image: "inngest-serverless"
    type:  "docker"
    memory: 128 // this doesn't need much ram
  }
}
```

You should commit this file to your repository. With this saved, you can now deploy:

```
$ inngestctl actions deploy ./action.cue
```

You tell inngestctl the file to deploy, then Inngest inspects the configuration, validates
it, finds the docker image you specified and uploads it from your machine to our platform.

Then, your action is ready for you to use in your account!

To see more serverless examples, head to our serverless actions repository: https://github.com/inngest/serverless-actions/
