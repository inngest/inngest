import type { ComponentProps } from 'react';
import { RunDetailsV4 } from '@inngest/components/RunDetailsV4';

type Props = Pick<
  ComponentProps<typeof RunDetailsV4>,
  'getTrigger' | 'pollInterval'
> & {
  runID?: string;
  runs?: readonly { id: string }[];
};

// A deep link can target a run outside the current filters or loaded pages.
// The existing details component fetches that exact ID independently of the list.
export function LinkedRunDetails({
  runID,
  runs,
  getTrigger,
  pollInterval,
}: Props) {
  if (!runID || runs?.some((run) => run.id === runID)) return null;

  return (
    <div className="border-subtle mx-4 mt-4 overflow-hidden rounded border">
      <RunDetailsV4
        standalone={false}
        runID={runID}
        getTrigger={getTrigger}
        pollInterval={pollInterval}
      />
    </div>
  );
}
