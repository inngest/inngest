# Storybook data

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
