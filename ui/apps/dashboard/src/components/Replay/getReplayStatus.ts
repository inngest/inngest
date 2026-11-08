import { ReplayStatus } from '@inngest/components/types/replay';

import type { GetReplayQuery } from '@/gql/graphql';

export function getReplayStatus(
  replay: Pick<GetReplayQuery['environment']['replay'], 'endedAt' | 'error'>,
): ReplayStatus {
  if (!replay.endedAt) {
    return ReplayStatus.Created;
  }

  return replay.error != null ? ReplayStatus.Failed : ReplayStatus.Ended;
}
