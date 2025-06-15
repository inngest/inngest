import { StepHistory } from './StepHistory';

const data = [
  {
    dateStarted: new Date('2025-06-14T12:07:00Z'),
    status: 'COMPLETED',
    tagCount: 1,
  },
  {
    dateStarted: new Date('2025-06-12T08:12:00Z'),
    status: 'FAILED',
    tagCount: 6,
  },
  {
    dateStarted: new Date('2025-06-11T12:32:00Z'),
    status: 'RUNNING',
    tagCount: 20,
  },
];

export const History = () => (
  <div className="flex w-full flex-col gap-2">
    {data.map((item) => (
      <StepHistory {...item} />
    ))}
  </div>
);
