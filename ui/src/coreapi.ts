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
      pendingSteps
      waitingFor {
        expiryTime
        eventName
        expression
      }
      event {
        id
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
            expiryTime
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

export const FUNCTIONS = gql`
  query GetFunctions {
    functions {
      id
      name
      triggers {
        type
        value
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
