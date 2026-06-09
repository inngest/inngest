// Local types for the scoring fields we select. Replace with imports from
// `@/gql/graphql` once the cloud schema is deployed and codegen picks them up.

export type ScoreKind = 'NUMERIC' | 'BOOLEAN';

export type ScoreName = {
  name: string;
  kind: ScoreKind;
};

export type ScoreBucket = {
  bucketStart: string;
  p50: number | null;
  p90: number | null;
  p99: number | null;
  trueCount: number | null;
  falseCount: number | null;
};

export type ScoreSeries = {
  scoreName: string;
  kind: ScoreKind;
  bucketSeconds: number;
  buckets: ScoreBucket[];
};

export type ScoreNamesResult = {
  scoreNames: ScoreName[];
};

export type ScoreTimeSeriesResult = {
  scoreTimeSeries: ScoreSeries[];
};
