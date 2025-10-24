import { Callout, CodeGroup, Properties, Property, Row, Col, VersionBadge, GuideSelector, GuideSection } from "src/shared/Docs/mdx";

export const hidePageSidebar = true;


# Inngest Errors

Inngest automatically handles errors and retries for you. You can use standard errors or use included Inngest errors to control how Inngest handles errors.

<GuideSelector
  options={[
    { key: "typescript", title: "TypeScript" },
    { key: "go", title: "Go" },
    { key: "python", title: "Python" },
  ]}>


## Standard errors

<GuideSection show="typescript">

All `Error` objects are handled by Inngest and [retried automatically](/docs/features/inngest-functions/error-retries/retries). This includes all standard errors like `TypeError` and custom errors that extend the `Error` class. You can throw errors in the function handler or within a step.

```typescript
export default inngest.createFunction(
  { id: "import-item-data" },
  { event: "store/import.requested" },
  async ({ event }) => {

    // throwing a standard error
    if (!event.itemId) {
      throw new Error("Item ID is required");
    }

    // throwing an error within a step
    const item = await step.run('fetch-item', async () => {
      const response = await fetch(`https://api.ecommerce.com/items/${event.itemId}`);
      if (response.status === 500) {
        throw new Error("Failed to fetch item from ecommerce API");
      }
      // ...
    });
  }
);
```
</GuideSection>

<GuideSection show="python">

All thrown Errors are handled by Inngest and [retried automatically](/docs/features/inngest-functions/error-retries/retries). This includes all standard errors like `ValueError` and custom errors that extend the `Exception` class. You can throw errors in the function handler or within a step.

```python
@client.create_function(
    fn_id="import-item-data",
    retries=0,
    trigger=inngest.TriggerEvent(event="store/import.requested"),
)
async def fn_async(ctx: inngest.Context) -> None:

    def foo() -> None:
        raise ValueError("foo")

    # a retry will be attempted
    await ctx.step.run("foo", foo)
```
</GuideSection>

<GuideSection show="go">

All Errors returned by your Inngest Functions are handled by Inngest and [retried automatically](/docs/features/inngest-functions/error-retries/retries). 

```go
import (
  "github.com/inngest/inngestgo"
  "github.com/inngest/inngestgo/step"
)

// Register the function
inngestgo.CreateFunction(
    client,
    inngestgo.FunctionOpts{
        ID: "send-user-email",
    },
    inngestgo.EventTrigger("user/created", nil),
    func(ctx context.Context, input inngestgo.Input[UserCreatedEvent]) (any, error) {
        // Run a step which emails the user.  This automatically retries on error.
        // This returns the fully typed result of the lambda.
        result, err := step.Run(ctx, "on-user-created", func(ctx context.Context) (bool, error) {
            // Run any code inside a step.
            result, err := emails.Send(emails.Opts{})
            return result, err
        })
        if err != nil {
            // This step retried 5 times by default and permanently failed.
            return nil, err
        }

        return result, nil
    },
)
```
</GuideSection>

## Prevent any additional retries

<GuideSection show="typescript">

Use `NonRetriableError` to prevent Inngest from retrying the function _or_ step. This is useful when the type of error is not expected to be resolved by a retry, for example, when the error is caused by an invalid input or when the error is expected to occur again if retried.

```typescript
import { NonRetriableError } from "inngest";

export default inngest.createFunction(
  { id: "mark-store-imported" },
  { event: "store/import.completed" },
  async ({ event }) => {
    try {
      const result = await database.updateStore(
        { id: event.data.storeId },
        { imported: true }
      );
      return result.ok === true;
    } catch (err) {
      // Passing the original error via `cause` enables you to view the error in function logs
      throw new NonRetriableError("Store not found", { cause: err });
    }
  }
);
```

### Parameters

```ts
new NonRetriableError(message: string, options?: { cause?: Error }): NonRetriableError
```

<Properties>
  <Property name="message" type="string" required>
    The error message.
  </Property>
  <Property name="options" type="object">
    <Properties nested={true} collapse={true}>
      <Property name="cause" type="Error">
        The original error that caused the non-retriable error.
      </Property>
    </Properties>
  </Property>
</Properties>

</GuideSection>

<GuideSection show="python">

Use `NonRetriableError` to prevent Inngest from retrying the function _or_ step. This is useful when the type of error is not expected to be resolved by a retry, for example, when the error is caused by an invalid input or when the error is expected to occur again if retried.

```python
@client.create_function(
    fn_id="import-item-data",
    retries=0,
    trigger=inngest.TriggerEvent(event="store/import.requested"),
)
async def fn_async(ctx: inngest.Context) -> None:
    def step_1() -> None:
        raise inngest.NonRetriableError("non-retriable-step-error")

    ctx.step.run("step_1", step_1)
``` 

</GuideSection>

<GuideSection show="go">

Use `inngestgo.NoRetryError` to prevent Inngest from retrying the function. This is useful when the type of error is not expected to be resolved by a retry, for example, when the error is caused by an invalid input or when the error is expected to occur again if retried.

```go
import (
  "github.com/inngest/inngestgo"
  "github.com/inngest/inngestgo/step"
)

// Register the function
inngestgo.CreateFunction(
    client,
    inngestgo.FunctionOpts{
        ID: "send-user-email",
    },
    inngestgo.EventTrigger("user/created", nil),
    func(ctx context.Context, input inngestgo.Input[UserCreatedEvent]) (any, error) {
        // Run a step which emails the user.  This automatically retries on error.
        // This returns the fully typed result of the lambda.
        result, err := step.Run(ctx, "on-user-created", func(ctx context.Context) (bool, error) {
            // Run any code inside a step.
            result, err := emails.Send(emails.Opts{})
            return result, err
        })
        if err != nil {
            // This step retried 5 times by default and permanently failed.
            // we return a NoRetryError to prevent Inngest from retrying the function
            return nil, inngestgo.NoRetryError(err)
        }

        return result, nil
    },
)
```

</GuideSection>

## Retry after a specific period of time

<GuideSection show="typescript">

Use `RetryAfterError` to control when Inngest should retry the function or step. This is useful when you want to delay the next retry attempt for a specific period of time, for example, to more gracefully handle a race condition or backing off after hitting an API rate limit.

If `RetryAfterError` is not used, Inngest will use [the default retry backoff policy](https://github.com/inngest/inngest/blob/main/pkg/backoff/backoff.go#L10-L22).

```typescript
inngest.createFunction(
  { id: "send-welcome-sms" },
  { event: "app/user.created" },
  async ({ event, step }) => {
    const { success, retryAfter } = await twilio.messages.create({
      to: event.data.user.phoneNumber,
      body: "Welcome to our service!",
    });

    if (!success && retryAfter) {
      throw new RetryAfterError("Hit Twilio rate limit", retryAfter);
    }
  }
);
```

### Parameters

```ts
new RetryAfterError(
  message: string,
  retryAfter: number | string | date,
  options?: { cause?: Error }
): RetryAfterError
```

<Properties>
  <Property name="message" type="string" required>
    The error message.
  </Property>
  <Property name="retryAfter" type="number | string | date" required>
    The specified time to delay the next retry attempt. The following formats are accepted:

    * `number` - The number of **milliseconds** to delay the next retry attempt.
    * `string` - A time string, parsed by the [ms](https://npm.im/ms) package, such as `"30m"`, `"3 hours"`, or `"2.5d"`.
    * `date` - A `Date` object.
  </Property>
  <Property name="options" type="object">
    <Properties nested={true} collapse={true}>
      <Property name="cause" type="Error">
        The original error that caused the non-retriable error.
      </Property>
    </Properties>
  </Property>
</Properties>

</GuideSection>

<GuideSection show="python">

Use `RetryAfterError` to control when Inngest should retry the function or step. This is useful when you want to delay the next retry attempt for a specific period of time, for example, to more gracefully handle a race condition or backing off after hitting an API rate limit.

If `RetryAfterError` is not used, Inngest will use [the default retry backoff policy](https://github.com/inngest/inngest/blob/main/pkg/backoff/backoff.go#L10-L22).

```python
@client.create_function(
    fn_id="import-item-data",
    retries=0,
    trigger=inngest.TriggerEvent(event="store/import.requested"),
)
async def fn_async(ctx: inngest.Context) -> None:
  def step_1() -> None:
      raise inngest.RetryAfterError("rate-limit-hit", 1000) # delay in milliseconds

  ctx.step.run("step_1", step_1)
``` 

### Parameters

```python
RetryAfterError(
  message: typing.Optional[str],
  retry_after: typing.Union[int, datetime.timedelta, datetime.datetime],
) -> None
```

<Properties>
  <Property name="message" type="string" required>
    The error message.
  </Property>
  <Property name="retry_after" type="int | datetime.timedelta | datetime.datetime" required>
    The specified time to delay the next retry attempt. The following formats are accepted:

    * `int` - The number of **milliseconds** to delay the next retry attempt.
    * `datetime.timedelta` - A time delta object, such as `datetime.timedelta(seconds=30)`.
    * `datetime.datetime` - A `datetime` object.
  </Property>
</Properties>

</GuideSection>

<GuideSection show="go">

Use `RetryAtError` to control when Inngest should retry the function or step. This is useful when you want to delay the next retry attempt for a specific period of time, for example, to more gracefully handle a race condition or backing off after hitting an API rate limit.

If `RetryAtError` is not used, Inngest will use [the default retry backoff policy](https://github.com/inngest/inngest/blob/main/pkg/backoff/backoff.go#L10-L22).

```go
import (
  "github.com/inngest/inngestgo"
  "github.com/inngest/inngestgo/step"
)

// Register the function
inngestgo.CreateFunction(
    client,
    inngestgo.FunctionOpts{
        ID: "send-user-email",
    },
    inngestgo.EventTrigger("user/created", nil),
    func(ctx context.Context, input inngestgo.Input[UserCreatedEvent]) (any, error) {
        // Run a step which emails the user.  This automatically retries on error.
        // This returns the fully typed result of the lambda.
        result, err := step.Run(ctx, "on-user-created", func(ctx context.Context) (bool, error) {
            // Run any code inside a step.
            result, err := emails.Send(emails.Opts{})
            return result, err
        })
        if err != nil {
            // This step retried 5 times by default and permanently failed.
            // We delay the next retry attempt by 5 hours
            return nil, inngestgo.RetryAtError(err, time.Now().Add(5*time.Hour))
        }

        return result, nil
    },
)
```

</GuideSection>

<GuideSection show="typescript">

## Step errors <VersionBadge version="v3.12.0+" />

After a step exhausts all of its retries, it will throw a `StepError` which can be caught and handled in the function handler if desired.

<CodeGroup>
```ts {{ title: "try/catch" }}
inngest.createFunction(
  { id: "send-weather-forecast" },
  { event: "weather/forecast.requested" },
  async ({ event, step }) => {
    let data;

    try {
      data = await step.run('get-public-weather-data', async () => {
        return await fetch('https://api.weather.com/data');
      });
    } catch (err) {
      // err will be an instance of StepError
      // Handle the error by recovering with a different step
      data = await step.run('use-backup-weather-api', async () => {
        return await fetch('https://api.stormwaters.com/data');
      });
    }
    // ...
  }
);
```
```ts {{ title: "Chaining with .catch()" }}
inngest.createFunction(
  { id: "send-weather-forecast" },
  { event: "weather/forecast.requested" },
  async ({ event, step }) => {

    const data = await step
      .run('get-public-weather-data', async () => {
        return await fetch('https://api.example.com/data');
      })
      .catch((err) => {
        // err will be an instance of StepError
        // Recover with a chained step
        return step.run("use-backup-weather-api", () => {
          return await fetch('https://api.stormwaters.com/data');
        });
      });
  }
);
```
```ts {{ title: "Ignoring and logging the error" }}
inngest.createFunction(
  { id: "send-weather-forecast" },
  { event: "weather/forecast.requested" },
  async ({ event, step }) => {

    const data = await step
      .run('get-public-weather-data', async () => {
        return await fetch('https://api.example.com/data');
      })
      // This will swallow the error and log it if it's non critical
      .catch((err) => logger.error(err));
  }
);
```
</CodeGroup>

<Callout>
  Support for handling step errors is available in the Inngest TypeScript SDK starting from version **3.12.0**. Prior to this version, wrapping a step in try/catch will not work correctly.
</Callout>

</GuideSection>

<GuideSection show="python">

## Step errors

After a step exhausts all of its retries, it will throw a `StepError` which can be caught and handled in the function handler if desired.

```python
@client.create_function(
    fn_id="import-item-data",
    retries=0,
    trigger=inngest.TriggerEvent(event="store/import.requested"),
)
async def fn_async(ctx: inngest.Context) -> None:
    def foo() -> None:
        raise ValueError("foo")

    try:
        ctx.step.run("foo", foo)
    except inngest.StepError:
        raise MyError("I am new")
``` 

</GuideSection>

<GuideSection show="typescript">

## Attempt counter

The current attempt number is passed in as input to the function handler. `attempt` is a zero-index number that increments for each retry. The first attempt will be `0`, the second `1`, and so on. The number is reset after a successfully executed step.

```ts
inngest.createFunction(
  { id: "generate-summary" },
  { event: "blog/post.created" },
  async ({ attempt }) => {
    // `attempt` is the zero-index attempt number

    await step.run('call-llm', async () => {
      if (attempt < 2) {
        // Call OpenAI's API two times
      } else {
        // After two attempts to OpenAI, try a different LLM, for example, Mistral
      }
    });
  }
);
```


## Stack traces

When calling functions that return Promises, await the Promise to ensure that the stack trace is preserved. This applies to functions executing in different cycles of the event loop, for example, when calling a database or an external API. This is especially useful when debugging errors in production.

<CodeGroup>
  ```ts {{ title: "Returning Promise" }}
  inngest.createFunction(
    { id: "update-recent-usage" },
    { event: "app/update-recent-usage" },
    async ({ event, step }) => {
      // ...
      await step.run("update in db", () => doSomeWork(event.data));
      // ...
    }
  );
  ```
  ```ts {{ title: "Awaiting Promise" }}
  inngest.createFunction(
    { id: "update-recent-usage" },
    { event: "app/update-recent-usage" },
    async ({ event, step }) => {
      // ...
      await step.run("update in db", async () => {
        return await doSomeWork(event.data);
      });
      // ...
    }
  );

  ```
</CodeGroup>

Please note that immediately returning the Promise will not include a pointer to the calling function in the stack trace. Awaiting the Promise will ensure that the stack trace includes the calling function.

</GuideSection>
</GuideSelector>