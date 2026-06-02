import { graphql } from '@/gql';

export const GetRunsDocument = graphql(`
  query GetRuns(
    $appIDs: [UUID!]
    $environmentID: ID!
    $startTime: Time!
    $endTime: Time
    $status: [FunctionRunStatus!]
    $timeField: RunsOrderByField!
    $functionSlug: String
    $functionRunCursor: String = null
    $celQuery: String = null
    $preview: Boolean = false
    $isDeferred: Boolean = null
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: {
          appIDs: $appIDs
          from: $startTime
          until: $endTime
          status: $status
          timeField: $timeField
          fnSlug: $functionSlug
          query: $celQuery
          isDeferred: $isDeferred
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
            deferredFrom {
              runID
              function {
                name
                slug
              }
            }
            id
            isDeferred
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
  }
`);

export const CountRunsDocument = graphql(`
  query CountRuns(
    $appIDs: [UUID!]
    $environmentID: ID!
    $startTime: Time!
    $endTime: Time
    $status: [FunctionRunStatus!]
    $timeField: RunsOrderByField!
    $functionSlug: String
    $celQuery: String = null
    $isDeferred: Boolean = null
  ) {
    environment: workspace(id: $environmentID) {
      runs(
        filter: {
          appIDs: $appIDs
          from: $startTime
          until: $endTime
          status: $status
          timeField: $timeField
          fnSlug: $functionSlug
          query: $celQuery
          isDeferred: $isDeferred
        }
        orderBy: [{ field: $timeField, direction: DESC }]
      ) {
        totalCount
      }
    }
  }
`);

export const AppFilterDocument = graphql(`
  query AppFilter($envSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
      }
    }
  }
`);
