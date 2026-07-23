import { graphql } from '@/gql';

// Bucket count for every trend query below (limit controls the number of
// time buckets a trend's range is divided into — see bucketDurationSeconds
// in pkg/applogic/dashboards/registry.go). Callers pass this as the
// $trendBucketLimit variable rather than each query hardcoding its own.
export const TREND_BUCKET_LIMIT = 21;

// Env-level AI Overview: all six registry entries, aliased per key in one
// round trip (the insightsMetric field returns one result per call — this
// is the aliasing pattern documented on the field itself, same as
// usage/metrics). Top-functions rankings are env-wide only, so they omit
// $functionIDs. See pkg/applogic/dashboards (backend) for the registry.
//
// insightsMetric returns the same generic InsightsResponse shape as the
// free-form Insights query (columns/rows/query) rather than a
// shape-specific union — see toScalarValues/toTrendPoints/
// toDimensionedTrendPoints/toListItems in ./types, which reconstruct the
// scalar/time-series/list shapes these components expect from that table.
export const InsightsOverviewMetricsDocument = graphql(`
  query InsightsOverviewMetrics(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $range: TimeRangeInput!
    $trendBucketLimit: Int
  ) {
    headline: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_headline"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    tokenTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_token_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    modelDistribution: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_model_distribution"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    runsByModel: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_runs_by_model"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    runVolumeTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_run_volume_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    latencyByModel: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_latency_by_model"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    latencyTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_latency_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    topFunctionsByRuns: insightsMetric(
      workspaceID: $workspaceID
      key: "ai_top_functions_by_runs"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    topFunctionsByCost: insightsMetric(
      workspaceID: $workspaceID
      key: "ai_top_functions_by_cost"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    mostExpensiveRuns: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_most_expensive_runs"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    mostExpensiveSteps: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_most_expensive_steps"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    mostExpensiveSessions: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_most_expensive_sessions"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    costPerRunTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_avg_cost_per_run_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    costPerSessionTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_cost_per_session_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    slowRuns: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_slow_runs"
      range: $range
      limit: 5
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

// Function-level AI tab: the subset of registry entries that make sense
// scoped to one function (no top-functions ranking — that's redundant when
// already scoped to a single function).
export const InsightsFunctionMetricsDocument = graphql(`
  query InsightsFunctionMetrics(
    $workspaceID: ID!
    $functionIDs: [ID!]
    $range: TimeRangeInput!
    $trendBucketLimit: Int
  ) {
    headline: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_headline"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    tokenTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_token_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    avgCostPerRunTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_avg_cost_per_run_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    tokenTrendByModel: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_token_trend_by_model"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    modelDistribution: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_model_distribution"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    runsByModel: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_runs_by_model"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    runVolumeTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_run_volume_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    latencyByModel: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_latency_by_model"
      range: $range
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    latencyTrend: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_latency_trend"
      range: $range
      limit: $trendBucketLimit
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    mostExpensiveRuns: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_most_expensive_runs"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    mostExpensiveSteps: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_most_expensive_steps"
      range: $range
      limit: 5
    ) {
      query
      columns {
        name
      }
      rows {
        values
      }
    }
    slowRuns: insightsMetric(
      workspaceID: $workspaceID
      functionIDs: $functionIDs
      key: "ai_slow_runs"
      range: $range
      limit: 5
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
