import { gql } from "graphql-request";

export const EVENTS_STREAM = gql`
  query GetEventsStream {
    events(query: {}) {
      id
      name
      createdAt
      status
      totalRuns
    }
  }
`;

export const FUNCTIONS_STREAM = gql`
  query GetFunctionsStream {
    functionRuns(query: {}) {
      id
      status
      startedAt
      pendingSteps
      name
      event {
        id
      }
    }
  }
`;

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
        waitingFor {
          waitUntil
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
      pendingSteps
      waitingFor {
        waitUntil
        eventName
        expression
      }
      event {
        raw
      }
      timeline {
        __typename
        ... on StepEvent {
          stepType: type
          createdAt
          output
          name
          waitingFor {
            waitUntil
            eventName
            expression
          }
        }
        ... on FunctionEvent {
          functionType: type
          createdAt
          output
        }
      }
    }
  }
`;
