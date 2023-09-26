# Storybook data

## Functions

```ts
inngest.createFunction(
  { name: 'Cancels', cancelOn: [{ event: 'foo' }] },
  { event: 'foo' },
  async ({ step }) => {
    await step.sleep('1m');
  },
);
```

```ts
inngest.createFunction({ name: 'Parallel steps' }, { event: 'foo' }, async ({ step }) => {
  await step.run('a', () => {});
  await Promise.all([step.run('b1', () => {}), step.run('b2', () => {})]);
});
```

```ts
inngest.createFunction({ name: 'Succeeds with 2 steps' }, { event: 'foo' }, async ({ step }) => {
  await step.run('First step', () => {});
  await step.run('Second step', async () => {});
});
```

```ts
inngest.createFunction({ name: 'Waits for event' }, { event: 'foo' }, async ({ step }) => {
  await step.waitForEvent('foo', '1m');
});
```

## Getting data

To get data, run this GraphQL query using the run ID:

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
