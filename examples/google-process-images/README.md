# Process images with Google Cloud and Sharp

<!-- Insert a short summary of the function. It should be no longer than a single paragraph -->
Respond to a `user/hello` event and output a `"Hello, [name]!"` message.

When a `user/profile.photo.uploaded` event is received, check that the uploaded image is safe using Google's Cloud Vision API, then process the images to create a variety of sizes.

<!-- Define a flowchart to visually show how the function will work -->
<!-- https://mermaid.live/ is a great tool for this, and docs are at https://mermaid-js.github.io/mermaid/#/flowchart -->
```mermaid
graph LR
Source[Your app] -->|"user/profile.photo.uploaded<br>'url':'https://.../photo.jpg'"| Inngest(Inngest)
Inngest -->|Triggers| Safety(steps/safety-check)
Safety -->|Not safe| Alert[steps/alert]
Safety -->|Safe| Process[steps/process]

classDef in fill:#4636f5,color:white,stroke:#4636f5;
classDef inngest fill:white,color:black,stroke:#4636f5,stroke-width:3px;
classDef out fill:white,color:black,stroke:#4636f5,stroke-dasharray: 5 5,stroke-width:3px;

class Source in;
class Inngest,Safety inngest;
class Alert,Process out;
```

<!-- To go along with the visual diagram, you can optionally add some numbered steps here to show the same flow -->
<!-- This may not always be required or appropriate, e.g. if there are some async actions happening -->
1. `user/profile.photo.uploaded` event is received
2. ➡️ Run [steps/safety-check](steps/safety-check) to check image safety
3. If image **is** deemed safe:
   - ✅➡️ Run [steps/process](steps/process) to create different image sizes and upload them to Google Cloud Storage
4. If image **is not** deemed safe:
   - ⚠️➡️ Run [steps/alert](steps/alert) to warn that the user has uploaded something unsafe

## Contents

<!-- A table of contents for your example, covering a few key areas -->
- [Usage](#usage)
- [Configuration](#configuration)
- [Code](#code)
- [Triggering the function](#triggering-the-function)

## Usage

<!-- A quick view of how to get started with the template. -->
<!-- The CLI can guide them -->
Use this quickstart with a single CLI command to get started! The CLI will then guide you through running, testing, and deploying to [Inngest Cloud](https//inngest.com/sign-up?ref=github-example).

```sh
npx inngest-cli init --template github.com/inngest/inngest#examples/google-process-images
```

Next, check out how to [👉 trigger the function](#triggering-the-function).

## Configuration

<!-- An annotated version of the `inngest.json|cue` file to help the user firm up the understanding of how the config works.-->

Below is the annotated function definition (found at [inngest.json](/inngest.json)) to show how the above is defined in config.

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/inngest/inngest/ad725bb6ca2b3846d412beb6ea8046e27a233154/schema.json",
  "name": "Process new profile photos with Google",
  "description": "Use the Google Cloud Vision API and Sharp to check images are safe and convert them to a variety of sizes.",
  "tags": ["typescript", "google"],
  "id": "free-doe-5f3107",
  "triggers": [
    {
      // When this event is received, we'll trigger our function
      "event": "user/profile.photo.uploaded",
      "definition": {
        "format": "cue",
        "synced": false,
        "def": "file://./events/user-profile-photo-uploaded.cue"
      }
    }
  ],
  "steps": {
    /**
     * Safety Check is the first step to run. It doesn't define an `after`
     * block, so will default to `{"step":"$trigger"}`.
     */
    "safety-check": {
      "id": "safety-check",
      "name": "Safety Check",
      "path": "file://./steps/safety-check",
      "runtime": {"type": "docker"}
    },
    /**
     * Process is one of the two steps that can run after the Safety Check. It
     * will run after that step if the output is that the image is deemed safe.
     */
    "process": {
      "id": "process",
      "name": "Process Images",
      "path": "file://./steps/process",
      "runtime": { "type": "docker" },
      "after": [
        {
          "step": "safety-check",
          "if": "steps['safety-check'].body.isSafe == true"
        }
      ]
    },
    /**
     * Alert is the other step that can run after the Safety Check. It will run
     * after that step if the output is that the image is deemed unsafe.
     */
    "alert": {
      "id": "alert",
      "name": "Alert",
      "path": "file://./steps/alert",
      "runtime": { "type": "docker" },
      "after": [
        {
          "step": "safety-check",
          "if": "steps['safety-check'].body.isSafe != true"
        }
      ]
    }
  }
}

```

## Code

This function has only a single step: `steps/hello`, which is triggered by the `user/hello` event.

<!-- A brief summary of where to find the various steps in the code and any other interesting configuration -->
- ➡️ [**steps/safety-check/**](steps/safety-check)
  > Using the `url` found in the event, this step will pass it to the Google Cloud Vision API to see if it is deemed safe. If the image is safe, `isSafe: true` will be passed as output to the next step. Otherwise, we'll pass `isSafe: false`.
- ➡️ [**steps/process/**](steps/process)
  > This step will run after [steps/safety-check](steps/safety-check) if the output is `isSafe: true`. It will stream the `url` from the event, pipe it to an image resizer, then pipe it to Google Cloud Storage. It outputs the created thumbnail URLs.
- ➡️ [**steps/alert/**](steps/alert)
  > If [steps/safety-check](steps/safety-check) returned `isSafe: false`, this step will run and output the user that uploaded the unsafe image. You could use this to push a notification to moderators or to flag the account for review.

## Triggering the function

<!-- Instructions for how the user should trigger the function from their infrastructure (or source) -->

Let's imagine a JavaScript application using the [Inngest JS SDK](https://github.com/inngest/inngest-js#readme).

In your `/signup` endpoint, you could add the following code:

```js
import { Inngest } from "inngest";

// POST myapp.com/signup
export default function signup(req, res) {
  const user = await createUser(req.body.email, req.body.password);

  // Send an event to Inngest
  // You can get a Source Key from the sources section of the Inngest app
  const inngest = new Inngest(process.env.INNGEST_SOURCE_API_KEY);
  await inngest.send({
    name: "app/user.signup",
    data: { city: req.body.city /* e.g. "Detroit" */ },
    user: {
      external_id: user.id,
      email: user.email,
    },
  });

  res.redirect("/app")
}
```
