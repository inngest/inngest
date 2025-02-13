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
        function {
          name
        }
        id
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
        name
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
      method
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

export const GET_APP = gql`
  query GetApp($id: UUID!) {
    app(id: $id) {
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
      method
      functions {
        name
        id
        concurrency
        config
        slug
        url
        triggers {
          type
          value
        }
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
  query GetTriggersStream($limit: Int!, $after: ID, $before: ID, $includeInternalEvents: Boolean!) {
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

export const INVOKE_FUNCTION = gql`
  mutation InvokeFunction($functionSlug: String!, $data: Map, $user: Map) {
    invokeFunction(data: $data, functionSlug: $functionSlug, user: $user)
  }
`;

export const CANCEL_RUN = gql`
  mutation CancelRun($runID: ULID!) {
    cancelRun(runID: $runID) {
      id
    }
  }
`;

export const RERUN = gql`
  mutation Rerun($runID: ULID!) {
    rerun(runID: $runID)
  }
`;

export const RERUN_FROM_STEP = gql`
  mutation RerunFromStep($runID: ULID!, $fromStep: RerunFromStepInput!) {
    rerun(runID: $runID, fromStep: $fromStep)
  }
`;

export const GET_RUNS = gql`
  query GetRuns(
    $appIDs: [UUID!]
    $startTime: Time!
    $status: [FunctionRunStatus!]
    $timeField: RunsV2OrderByField!
    $functionRunCursor: String = null
    $celQuery: String = null
  ) {
    runs(
      filter: {
        appIDs: $appIDs
        from: $startTime
        status: $status
        timeField: $timeField
        query: $celQuery
      }
      orderBy: [{ field: $timeField, direction: DESC }]
      after: $functionRunCursor
    ) {
      edges {
        node {
          app {
            externalID
            name
          }
          cronSchedule
          eventName
          function {
            name
            slug
          }
          id
          isBatch
          queuedAt
          endedAt
          startedAt
          status
          hasAI
        }
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
    }
  }
`;

export const COUNT_RUNS = gql`
  query CountRuns(
    $startTime: Time!
    $status: [FunctionRunStatus!]
    $timeField: RunsV2OrderByField!
  ) {
    runs(
      filter: { from: $startTime, status: $status, timeField: $timeField }
      orderBy: [{ field: $timeField, direction: DESC }]
    ) {
      totalCount
    }
  }
`;

export const TRACE_DETAILS_FRAGMENT = gql`
  fragment TraceDetails on RunTraceSpan {
    name
    status
    attempts
    queuedAt
    startedAt
    endedAt
    isRoot
    outputID
    spanID
    stepID
    stepOp
    stepInfo {
      __typename
      ... on InvokeStepInfo {
        triggeringEventID
        functionID
        timeout
        returnEventID
        runID
        timedOut
      }
      ... on SleepStepInfo {
        sleepUntil
      }
      ... on WaitForEventStepInfo {
        eventName
        expression
        timeout
        foundEventID
        timedOut
      }
      ... on RunStepInfo {
        type
      }
    }
  }
`;

export const GET_RUN = gql`
  query GetRun($runID: String!) {
    run(runID: $runID) {
      function {
        app {
          name
        }
        id
        name
        slug
      }
      trace {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
          }
        }
      }
      hasAI
    }
  }
`;

export const GET_TRACE_RESULT = gql`
  query GetTraceResult($traceID: String!) {
    runTraceSpanOutputByID(outputID: $traceID) {
      input
      data
      error {
        message
        name
        stack
      }
    }
  }
`;

export const GET_TRIGGER = gql`
  query GetTrigger($runID: String!) {
    runTrigger(runID: $runID) {
      IDs
      payloads
      timestamp
      eventName
      isBatch
      batchID
      cron
    }
  }
`;

export const GET_WORKER_CONNECTIONS = gql`
  query GetWorkerConnections(
    $appID: UUID!
    $startTime: Time
    $status: [ConnectV1ConnectionStatus!]
    $timeField: ConnectV1WorkerConnectionsOrderByField!
    $cursor: String = null
    $orderBy: [ConnectV1WorkerConnectionsOrderBy!] = []
    $first: Int!
  ) {
    workerConnections(
      first: $first
      filter: { appIDs: [$appID], from: $startTime, status: $status, timeField: $timeField }
      orderBy: $orderBy
      after: $cursor
    ) {
      edges {
        node {
          id
          gatewayId
          instanceId
          workerIp
          app {
            id
          }
          connectedAt
          lastHeartbeatAt
          disconnectedAt
          disconnectReason
          status
          groupHash
          sdkLang
          sdkVersion
          sdkPlatform
          syncId
          buildId
          functionCount
          cpuCores
          memBytes
          os
        }
      }
      pageInfo {
        hasNextPage
        hasPreviousPage
        startCursor
        endCursor
      }
    }
  }
`;

export const COUNT_WORKER_CONNECTIONS = gql`
  query CountWorkerConnections($appID: UUID!, $status: [ConnectV1ConnectionStatus!]) {
    workerConnections(
      filter: { appIDs: [$appID], status: $status, timeField: CONNECTED_AT }
      orderBy: [{ field: CONNECTED_AT, direction: DESC }]
    ) {
      totalCount
    }
  }
`;
