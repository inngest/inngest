import { graphql } from '@/gql';

export const ScoreNamesDocument = graphql(`
  query ScoreNames(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $filter: ScoreFilter!
  ) {
    scoreNames(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      filter: $filter
    ) {
      name
      kind
    }
  }
`);

export const ScoreTimeSeriesDocument = graphql(`
  query ScoreTimeSeries(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $filter: ScoreFilter!
    $scoreNames: [String!]
  ) {
    scoreTimeSeries(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      filter: $filter
      scoreNames: $scoreNames
    ) {
      scoreName
      kind
      buckets {
        bucketStart
        avg
        max
        p50
        p90
        p99
        trueCount
        falseCount
      }
    }
  }
`);
