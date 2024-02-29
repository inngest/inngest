import { gql } from 'graphql-request';

export const EVENT = gql`
  query GetEvent($id: ID!) {
    event(query: { eventId: $id }) {
      id
      name
      createdAt
      status
      pendingRuns
      raw
      functionRuns {
        id
        name
        status
        startedAt
        pendingSteps
        output
        waitingFor {
          expiryTime
          eventName
          expression
        }
      }
    }
  }
`;

export const FUNCTION_RUN = gql`
  query GetFunctionRun($id: ID!) {
    functionRun(query: { functionRunId: $id }) {
      id
      name
      status
      startedAt
      finishedAt
      output
      pendingSteps
      waitingFor {
        expiryTime
        eventName
        expression
      }
      function {
        triggers {
          type
          value
        }
      }
      event {
        id
        raw
      }
      batchID
      batchCreatedAt
      events {
        createdAt
        id
        name
        raw
      }
      history {
        attempt
        cancel {
          eventID
          expression
          userID
        }
        createdAt
        functionVersion
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
        invokeFunction {
          eventID
          functionID
          correlationID
          timeout
        }
        invokeFunctionResult {
          eventID
          timeout
          runID
        }
      }
    }
  }
`;

export const FUNCTIONS = gql`
  query GetFunctions {
    functions {
      id
      slug
      name
      triggers {
        type
        value
      }
      app {
        name
      }
      url
    }
  }
`;

export const APPS = gql`
  query GetApps {
    apps {
      id
      name
      sdkLanguage
      sdkVersion
      framework
      url
      error
      connected
      functionCount
      autodiscovered
      functions {
        name
        id
        concurrency
        config
        slug
        url
      }
    }
  }
`;

export const ADD_APP = gql`
  mutation CreateApp($input: CreateAppInput!) {
    createApp(input: $input) {
      url
    }
  }
`;

export const UPDATE_APP = gql`
  mutation UpdateApp($input: UpdateAppInput!) {
    updateApp(input: $input) {
      url
      id
    }
  }
`;

export const DELETE_APP = gql`
  mutation DeleteApp($id: String!) {
    deleteApp(id: $id)
  }
`;

export const TRIGGERS_STREAM = gql`
  query GetTriggersStream(
    $limit: Int!
    $after: Time
    $before: Time
    $includeInternalEvents: Boolean!
  ) {
    stream(
      query: {
        limit: $limit
        after: $after
        before: $before
        includeInternalEvents: $includeInternalEvents
      }
    ) {
      createdAt
      id
      inBatch
      trigger
      type
      runs {
        batchID
        events {
          id
        }
        id
        function {
          name
        }
      }
    }
  }
`;

export const FUNCTION_RUN_STATUS = gql`
  query GetFunctionRunStatus($id: ID!) {
    functionRun(query: { functionRunId: $id }) {
      id
      function {
        name
      }
      status
    }
  }
`;

export const FUNCTION_RUN_OUTPUT = gql`
  query GetFunctionRunOutput($id: ID!) {
    functionRun(query: { functionRunId: $id }) {
      id
      status
      output
    }
  }
`;

export const HISTORY_ITEM_OUTPUT = gql`
  query GetHistoryItemOutput($historyItemID: ULID!, $runID: ID!) {
    functionRun(query: { functionRunId: $runID }) {
      historyItemOutput(id: $historyItemID)
    }
  }
`;
