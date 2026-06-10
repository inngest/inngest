// Alias over the codegen'd query result so consumers don't repeat the
// indexed-access type. The schema types themselves live in `@/gql/graphql`.
import type { ScoreTimeSeriesQuery } from '@/gql/graphql';

export type ScoreSeries = ScoreTimeSeriesQuery['scoreTimeSeries'][number];
