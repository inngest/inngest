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

export const FUNCTION = gql`
  query GetFunction($functionSlug: String!) {
    functionBySlug(query: { functionSlug: $functionSlug }) {
      name
      id
      failureHandler {
        slug
      }
      concurrency
      config
      configuration {
        cancellations {
          event
          timeout
          condition
        }
        retries {
          value
          isDefault
        }
        priority
        eventsBatch {
          maxSize
          timeout
          key
        }
        concurrency {
          scope
          limit {
            value
            isPlanLimit
          }
          key
        }
        rateLimit {
          limit
          period
          key
        }
        debounce {
          period
          key
        }
        throttle {
          burst
          key
          limit
          period
        }
        singleton {
          key
          mode
        }
      }
      slug
      triggers {
        type
        value
        condition
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
      appVersion
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
      appVersion
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

export const INVOKE_FUNCTION = gql`
  mutation InvokeFunction(
    $functionSlug: String!
    $data: Map
    $user: Map
    $debugSessionID: ULID = null
    $debugRunID: ULID = null
  ) {
    invokeFunction(
      data: $data
      functionSlug: $functionSlug
      user: $user
      debugSessionID: $debugSessionID
      debugRunID: $debugRunID
    )
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
  mutation Rerun(
    $runID: ULID!
    $debugRunID: ULID = null
    $debugSessionID: ULID = null
  ) {
    rerun(
      runID: $runID
      debugRunID: $debugRunID
      debugSessionID: $debugSessionID
    )
  }
`;

export const RERUN_FROM_STEP = gql`
  mutation RerunFromStep(
    $runID: ULID!
    $fromStep: RerunFromStepInput!
    $debugRunID: ULID = null
    $debugSessionID: ULID = null
  ) {
    rerun(
      runID: $runID
      fromStep: $fromStep
      debugRunID: $debugRunID
      debugSessionID: $debugSessionID
    )
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
    $preview: Boolean = false
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
      preview: $preview
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
    $preview: Boolean = false
  ) {
    runs(
      filter: { from: $startTime, status: $status, timeField: $timeField }
      orderBy: [{ field: $timeField, direction: DESC }]
      preview: $preview
    ) {
      totalCount(preview: $preview)
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
    isUserland
    userlandSpan {
      spanName
      spanKind
      serviceName
      scopeName
      scopeVersion
      spanAttrs
      resourceAttrs
    }
    metadata {
      scope
      kind
      values
      updated_at
    }
    outputID
    debugRunID
    debugSessionID
    spanID
    stepID
    stepOp
    stepType
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
      ... on WaitForSignalStepInfo {
        signal
        timeout
        timedOut
      }
    }
  }
`;

export const GET_RUN = gql`
  query GetRun($runID: String!, $preview: Boolean) {
    run(runID: $runID) {
      function {
        app {
          name
        }
        id
        name
        slug
      }
      status
      trace(preview: $preview) {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
              childrenSpans {
                ...TraceDetails
              }
            }
          }
        }
      }
      hasAI
    }
  }
`;

export const GET_RUN_TRACE = gql`
  query GetRunTrace($runID: String!) {
    runTrace(runID: $runID) {
      ...TraceDetails
      childrenSpans {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
            }
          }
        }
      }
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
        cause
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
      filter: {
        appIDs: [$appID]
        from: $startTime
        status: $status
        timeField: $timeField
      }
      orderBy: $orderBy
      after: $cursor
    ) {
      edges {
        node {
          id
          gatewayId
          instanceId
          workerIp
          maxWorkerConcurrency
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
          appVersion
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
      totalCount
    }
  }
`;

export const COUNT_WORKER_CONNECTIONS = gql`
  query CountWorkerConnections(
    $appID: UUID!
    $startTime: Time!
    $status: [ConnectV1ConnectionStatus!]
  ) {
    workerConnections(
      filter: {
        appIDs: [$appID]
        from: $startTime
        status: $status
        timeField: CONNECTED_AT
      }
      orderBy: [{ field: CONNECTED_AT, direction: DESC }]
    ) {
      totalCount
    }
  }
`;

export const GET_EVENTS = gql`
  query GetEventsV2(
    $cursor: String
    $startTime: Time!
    $endTime: Time
    $celQuery: String = null
    $eventNames: [String!] = null
    $includeInternalEvents: Boolean = false
  ) {
    eventsV2(
      first: 50
      after: $cursor
      filter: {
        from: $startTime
        until: $endTime
        query: $celQuery
        eventNames: $eventNames
        includeInternalEvents: $includeInternalEvents
      }
    ) {
      edges {
        node {
          name
          id
          receivedAt
          runs {
            status
            id
            startedAt
            endedAt
            function {
              name
              slug
            }
          }
        }
      }
      totalCount
      pageInfo {
        hasNextPage
        endCursor
        hasPreviousPage
        startCursor
      }
    }
  }
`;

export const GET_EVENT = gql`
  query GetEventV2($eventID: ULID!) {
    eventV2(id: $eventID) {
      name
      id
      receivedAt
      idempotencyKey
      occurredAt
      version
      source {
        name
      }
    }
  }
`;
export const GET_EVENT_PAYLOAD = gql`
  query GetEventV2Payload($eventID: ULID!) {
    eventV2(id: $eventID) {
      raw
    }
  }
`;

export const GET_EVENT_RUNS = gql`
  query GetEventV2Runs($eventID: ULID!) {
    eventV2(id: $eventID) {
      name
      runs {
        status
        id
        startedAt
        endedAt
        function {
          name
          slug
        }
      }
    }
  }
`;

export const CREATE_DEBUG_SESSION = gql`
  mutation CreateDebugSession($input: CreateDebugSessionInput!) {
    createDebugSession(input: $input) {
      debugSessionID
      debugRunID
    }
  }
`;

export const DEBUG_RUN = gql`
  query GetDebugRun($query: DebugRunQuery!) {
    debugRun(query: $query) {
      debugTraces {
        ...TraceDetails
        childrenSpans {
          ...TraceDetails
          childrenSpans {
            ...TraceDetails
            childrenSpans {
              ...TraceDetails
            }
          }
        }
      }
    }
  }
`;

export const DEBUG_SESSION = gql`
  query GetDebugSession($query: DebugSessionQuery!) {
    debugSession(query: $query) {
      debugRuns {
        status
        queuedAt
        startedAt
        endedAt
        debugRunID
        tags
        versions
      }
    }
  }
`;
