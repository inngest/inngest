---
category: "Functions"
title: "Expressions"
slug: "functions/expressions"
order: 85
---

Functions can use expressions to conditionally continue to the next step.  This
allows you to write complex logic to manage your function execution.

Functions can have expressions in many places:

- **Trigger expressions** allow you to run your function only when event data matches specific
  conditions
- **Step expressions** add conditional checks that must pass before continuing to
  the next step
- **Asynchronous expressions** allow you to pause in between steps until a
  new event is received that match a specific condition.

Here's an example configuration file with all three expressions:

```json twoslash
{
  "name": "Send SMS Dispatch",
  "id": "my-function-ede40d",
  "triggers": [
    {
      "event": "api/response.received",
// @log: This trigger expression specifies that the function should only run if the incoming event's "data.status" field is 200.  Events that do not match will not trigger this function.
      "expression": "event.data.status == 200"
    }
  ],

  "steps": {
    "step-1": {
      "id": "step-1",
      "path": "file://./steps/send-sms-dispatch",
      "name": "Send SMS Dispatch",
      "runtime": {
        "type": "docker"
      }
    },

    "step-2": {
      "id": "step-2",
      "path": "file://./steps/my-second-step",
      "name": "Another step",
      "runtime": {
        "type": "docker"
      },
      "after": [
        {
          "step": "step-1",
// @log: This edge expression indicates that step-2 should only run if the the output of the previous step (step-1) contains the data body.email which matches 'hello@example.com'.  If the step doesn't respond with this data, step-2 will not run.
          "if": "steps['step-1'].body.email == 'hello@example.com'"
        }
      ]
    },

    "last": {
      "id": "last",
      "path": "file://./steps/last",
      "name": "Last step",
      "runtime": {
        "type": "docker"
      },
      "after": [
        {
          "step": "step-2",
          "async": {
            "event": "user/checkout",
            "ttl": "1h",
// @log: This async expression indicates that we should only continue to the last step when we receive a `user/checkout` event (`async`) which has the same user ID as the original event (`event`)
            "match": "async.user.id == event.user.id"
          }
        }
      ]
    }
  }
}
```


## Trigger expressions

Trigger expressions allow you to run your function based off of conditionally
checking the incoming event data.

#### Available data

- `event`: The event data.

## Step expressions

Edges can

**Available data**

## Asynchronous expressions
what

**Available data**

what

## Functions and helpers

what
