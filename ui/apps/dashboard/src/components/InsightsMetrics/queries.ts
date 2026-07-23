import { graphql } from '@/gql';

// Bucket count for every trend query below (limit controls the number of
// time buckets a trend's range is divided into — see bucketDurationSeconds
// in pkg/applogic/dashboards/registry.go). Callers pass this as the
// $trendBucketLimit variable rather than each query hardcoding its own.
export const TREND_BUCKET_LIMIT = 21;

// Every registry entry (env-level AI Overview, function-level AI tab, and
// usage/metrics elsewhere) resolves through this one field, varying only
// $key/$limit/$functionIDs — so callers issue one InsightsMetric request per
// widget instead of aliasing every key into a single combined round trip.
// This lets each card fetch, load, and error independently, and lets
// $functionIDs-independent keys (e.g. top-functions rankings) avoid
// recomputing when an unrelated widget's variables change.
//
// insightsMetric returns the same generic InsightsResponse shape as the
// free-form Insights query (columns/rows/query) rather than a
// shape-specific union — see toScalarValues/toTrendPoints/
// toDimensionedTrendPoints/toListItems in ./types, which reconstruct the
// scalar/time-series/list shapes these components expect from that table.
// See pkg/applogic/dashboards (backend) for the registry.
export const InsightsMetricDocument = graphql(`
  query InsightsMetric(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $key: String!
    $range: TimeRangeInput!
    $limit: Int
  ) {
    insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: $key
      range: $range
      limit: $limit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
  }
`);
