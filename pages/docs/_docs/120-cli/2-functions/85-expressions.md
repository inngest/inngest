---
category: "Functions"
title: "Expressions"
slug: "functions/expressions"
order: 85
hide: true
---

Functions can use expressions to conditionally continue to the next step.  This
allows you to write complex logic to manage your function execution.

Functions can have expressions in many places:

- **Trigger expressions** allow you to run your function only when event data matches specific
  conditions
- **Step expressions** add conditional checks that must pass before continuing to
  the next step
- **Asynchronous (async) expressions** allow you to pause in between steps until a
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
// @log: This step expression indicates that step-2 should only run if the the output of the previous step (step-1) contains the data body.email which matches 'hello@example.com'.  If the step doesn't respond with this data, step-2 will not run.
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

Trigger expressions allow you to conditionally run your function based off of
data within the event.

Note that all trigger expressions are ran prior to executing functions.  If an
event does not match the expression the function and its steps will not run and
you will not be charged.

**Available data**

The following variables are available within trigger expressions: 

- `event`: The event data.


**Examples**

- You only want to run a function if a GitHub CI check failed: <br />
  `event.data.workflow_job.status == 'failed'`
- You only want to send users a coupon code if their order was over $100, and
  the user ordered more than 1 product: <br />
  `event.data.amount >= 100 && size(event.data.items) > 1`


```json twoslash
{
  "name": "Create coupon for valued orders",
  "id": "my-function-ede40d",
  "triggers": [
    {
      "event": "shop/checkout.complete",
      "expression": "event.data.amount >= 100 && size(event.data.items) > 1"
    }
  ],
  "steps": {
    "coupon": {
      "id": "coupon",
      "path": "file://./steps/create-coupon",
      "name": "Create coupon",
      "runtime": {
        "type": "docker"
      }
    }
  }
}
```

## Step expressions

Step expressions allow you to conditionally run steps within a step function.

This pattern allows you to branch over individual steps of a function, instead
of writing one single complex step which has nested logic.  This provides
several benefits:

- Each step does exactly one thing, reducing complexity
- Steps can be handled in parallel
- Individual steps can be retried on intermittent errors

Step expressions have access to the event, plus the output of all previous
steps.

**Available data**

- `event`: The event data.
- `steps`: The output of previously completed steps, as a map keyed by step ID.
- `response`: A shortcut for the output of the parent step

**Examples**

- Only continuing if a step with the ID `checkStatus` returns "delinquent": <br />
  `steps["checkStatus"].body.status == "delinquent"`

```json twoslash
{
  "name": "Send SMS Dispatch",
  "id": "my-function-ede40d",
  "triggers": [
    {
      "event": "api/response.received",
    }
  ],

  "steps": {
    "checkStatus": {
      "id": "checkStatus",
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
          "if": "steps['checkStatus'].body.status == 'delinquent'"
	  // NOTE: This could be written as `response.body.status == 'delinquent'`,
	  // as `response` always represents the parent step that just ran.
        }
      ]
    }
  }
}
```

## Asynchronous (async) expressions

Asynchronous (async) expressions let you pause a running function, wait for a new event
to be received, then test the new event to match an expression before resuming
the function.

This allows you to build complex user journeys, approval logic, human-in-the-loop
steps and request-reply functionality without managing orchestration.

Async expressions have access to the original event data, all completed step data,
and the incoming async event's data.

**Available data**

- `event`: The event data for the event which triggered the function.
- `steps`: The output of previously completed steps, as a map keyed by step ID.
- `async`: The data of the new event which can resume the workflow.

**Examples**

- When a user adds an item to the cart, wait for the *same* user to check out
  before continuing: <br />
  `async.data.cart_id == event.data.cart_id`
- When a task is created, wait for the same task to be completed: <br />
  `async.data.task_id == event.data.task_id && async.data.action == 'completed'`

```json twoslash
{
  "name": "Wait for task completion",
  "id": "my-function-ede40d",
  "triggers": [
    { "event": "api/task.created" }
  ],

  "steps": {
    "complete": {
      "id": "complete",
      "path": "file://./steps/send-sms-dispatch",
      "name": "Send SMS Dispatch",
      "runtime": {
        "type": "docker"
      }
      "after": [
        {
          "step": "$trigger",
          "async": {
            "event": "api/task.updated",
            "ttl": "1w",
            "match": "async.data.task_id == event.data.task_id"
          }
        }
      ]
    }
  }
}
```

## Functions and helpers

While the expression language is not a fully featured programming language,
there are many helper functions available.

**Index**

`all`<br />
`contains`<br />
`date`<br />
`endsWith`<br />
`exists`<br />
`filter`<br />
`lowercase`<br />
`map`<br />
`matches`<br />
`now`<br />
`now_minus`<br />
`now_plus`<br />
`size`<br />
`startsWith`<br />
`uppercase`<br />

<br />

**`[].all`**

Checks that each item of an array matches a sub-expression.  To check that only
some elements match, use `exists`.

```
[1, 2, 3].all(x, x >= 1)
```

```
["some", "long", "words"].all(word, size(word) > 3)
```

<br />

**`contains`**

Checks whether a string contains data.  Examples:

```
"submarine".contains("sub") == true
```

```
event.data.name.contains(steps["my-step"].output)
```


<br />

**`date`**

Converts a string date into a timestamp for comparison.  Valid formats are
RFC3339, ISO8601, RFC1123, RFC822, RFC850, YYYY-MM-DD, Unix timestamps,
Millisecond timestamps, and unix dates.  Each format will be attempted until
a matching format is found.

Parsing a YYYY-MM-DD date:
```
date("2021-05-08") < now() == true
```

Parsing a millisecond unix timestamp:
```
date(1660678425172) < now() == true
```

Parsing a date within event data:
```
date(event.data.next_visit) > now_plus("24h")
```

<br />

**`endsWith`**

Checks whether a string ends with another string.  Examples:

```
"submarine".endsWith("marine") == true
```

```
event.data.email.endsWith("@gmail.com")
```

<br />

**`[].exists`**

Checks that at least one item of an array matches a sub-expression.  To check
that all elements match, use `all`.

```
[100, 195, 2599].exists(price, price > 999)
```

<br />

**`lowercase`**

Converts a UTF-8 string to lowercase

```
lowercase("SUBMARINE") == "submarine"
```
<br />

**`[].map`**

Iterates through an array and applies a function, returning the result.

```
[1, 2, 3].map(x, x * 2) == [2, 4, 6]
```

```
["some", "long", "words"].map(word, uppercase(word)) == ["SOME", "LONG", WORDS"]
```

<br />

**`[].filter`**

Returns all items of an array that match a sub-expression.

```
[12, 24, 199].filter(price, price < 100) == [12, 24]
```

<br />

**`lowercase`**

Converts a UTF-8 string to lowercase

```
lowercase("SUBMARINE") == "submarine"
```
<br />

**`[].map`**

Iterates through an array and applies a function, returning the result.

```
[1, 2, 3].map(x, x * 2) == [2, 4, 6]
```

```
["some", "long", "words"].map(word, uppercase(word)) == ["SOME", "LONG", WORDS"]
```


<br />

**`matches`**

Matches a string against a regular expression

```
"my long string".matches('^\w+$') == true
```

<br />

**`now`**

Returns the current time

```
now()
```

<br />

**`now_minus`**

Returns the current time minus a given duration.  Any duration greater than "h"
ignores DST changes and leap seconds.

Available durations:
- `ms`: milliseconds
- `s`: seconds
- `m`: minutes
- `h`: hours
- `d`: days (24 hours)
- `w`: weeks (7 * 24 hours)

```
now_minus("1m30s") // now minus 1m30s
```

```
now_minus("1w") // now minus 1 week 
```

<br />

**`now_plus`**

Returns the current time plus a given duration.  Any duration greater than "h"
ignores DST changes and leap seconds.

Available durations:
- `ms`: milliseconds
- `s`: seconds
- `m`: minutes
- `h`: hours
- `d`: days (24 hours)
- `w`: weeks (7 * 24 hours)

```
now_plus("12h30m") // now plus 12 hours 30 minutes
```

```
now_plus("1h") // now plus 1 hour 
```

<br />

**`size`**

Returns the size of the current data.  This returns the total number of items
within an array, or the length of a string.

```
size([0, 8, 2]) == 3
```

```
size("abc") == 3
```

<br />

**`startsWith`**

Checks whether a string starts with another string

```
"submarine".startsWith("sub") == true
```

```
"example@example.com".startsWith("example") == true
```

**`uppercase`**

Converts a UTF-8 string to uppercase

```
uppercase("submarine") == "SUBMARINE"
```
