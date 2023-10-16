# Storybook data

## Functions

```ts
inngest.createFunction(
  { name: 'Cancels', cancelOn: [{ event: 'foo' }] },
  { event: 'foo' },
  async ({ step }) => {
    await step.sleep('1m');
  }
);
```

```ts
inngest.createFunction(
  { name: 'Fails without steps', cancelOn: [{ event: 'foo' }] },
  { event: 'foo' },
  async ({ step }) => {
    throw new Error('oh no');
  }
);
```

```ts
inngest.createFunction(
  { name: 'Fails with preceding step', cancelOn: [{ event: 'foo' }] },
  { event: 'foo' },
  async ({ step }) => {
    await step.run('First step', () => {});

    await step.run('Second step', () => {
      throw new Error('oh no');
    });
  }
);
```

```ts
inngest.createFunction({ name: 'No steps' }, { event: 'foo' }, async ({ step }) => {});
```

```ts
inngest.createFunction({ name: 'Parallel steps' }, { event: 'foo' }, async ({ step }) => {
  await step.run('a', () => {});
  await Promise.all([step.run('b1', () => {}), step.run('b2', () => {})]);
});
```

```ts
inngest.createFunction({ name: 'Sleeps' }, { event: 'foo' }, async ({ step }) => {
  await step.sleep('10s');
});
```

```ts
inngest.createFunction({ name: 'Succeeds with 2 steps' }, { event: 'foo' }, async ({ step }) => {
  await step.run('First step', () => {});
  await step.run('Second step', async () => {});
});
```

```ts
inngest.createFunction(
  { name: 'Times out waiting for event' },
  { event: 'foo' },
  async ({ step }) => {
    await step.waitForEvent('bar', '10s');
  }
);
```

```ts
// Need to manually send the bar event to fulfill the waitForEvent.
inngest.createFunction({ name: 'Waits for event' }, { event: 'foo' }, async ({ step }) => {
  await step.waitForEvent('bar', '1m');
});
```

## Getting data

To get data, run the following GraphQL query using the run ID. You can find the GraphQL playground at http://localhost:8288/v0.

```gql
{
  functionRun(query: { functionRunId: "<run-id>" }) {
    history {
      attempt
      cancel {
        eventID
        expression
        userID
      }
      createdAt
      groupID
      id
      sleep {
        until
      }
      stepName
      type
      url
      waitForEvent {
        eventName
        expression
        timeout
      }
      waitResult {
        eventID
        timeout
      }
    }
  }
}
```

Then copy the `data.functionRun.history` array into a JSON file.
