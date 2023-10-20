import dayjs from 'dayjs';

import Table from '@/components/Table';
import { defaultTime, duration } from '@/utils/date';

const replays = [
  {
    name: 'Replay 1',
    status: 'Completed',
    startedAt: new Date('2023-10-18T12:00:00Z'),
    runsCount: 130,
  },
  {
    name: 'Replay 2',
    status: 'Running',
    startedAt: new Date('2023-10-20T12:00:00Z'),
    runsCount: 130,
  },
  {
    name: 'Replay 3',
    status: 'Failed',
    startedAt: new Date('2023-10-18T12:00:00Z'),
    runsCount: 130,
  },
];

type FunctionReplayPageProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};
export default function FunctionReplayPage({ params }: FunctionReplayPageProps) {
  const replaysInTableFormat = replays.map((replay) => {
    return {
      ...replay,
      startedAt: replay.startedAt.toLocaleString(),
      elapsed: duration(dayjs.duration(dayjs().diff(replay.startedAt))),
    };
  });

  return (
    <Table
      columns={[
        { key: 'status', className: 'w-14' },
        { key: 'name', label: 'Replay Name' },
        { key: 'startedAt', label: 'Started At' },
        { key: 'elapsed', label: 'Elapsed' },
        { key: 'runsCount', label: 'Total Runs' },
      ]}
      empty="You have no replays for this function."
      data={replaysInTableFormat}
    />
  );
}
