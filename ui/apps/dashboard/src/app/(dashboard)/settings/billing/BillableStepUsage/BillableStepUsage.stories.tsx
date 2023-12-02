import type { Meta, StoryObj } from '@storybook/react';

import { BaseWrapper } from '@/app/baseWrapper';
import { type TimeSeries } from '@/gql/graphql';

// import { BillableStepUsage } from './BillableStepUsage';

function createData({ month, year }: { month: number; year: number }): TimeSeries['data'] {
  const daysInMonth = new Date(year, month + 1, 0).getDate();

  const out: TimeSeries['data'] = [];
  for (let day = 1; day <= daysInMonth; day++) {
    out.push({
      time: new Date(year, month, day).toISOString(),
      value: Math.floor(generateRandomNumber(day) * 6000),
    });
  }

  return out;
}

/**
 * Deterministically generate a random number between 0 and 1.
 */
function generateRandomNumber(seed: number): number {
  let x = Math.sin(seed) * 10000;
  return x - Math.floor(x);
}

const Disable = () => <>disabled story</>;

// const meta: Meta<typeof BillableStepUsage> = {
const meta: Meta<typeof Disable> = {
  args: {
    // data: {
    //   prevMonth: createData({ month: 3, year: 2023 }),
    //   thisMonth: createData({ month: 4, year: 2023 }),
    // },
    includedStepCountLimit: 50_000,
  },
  decorators: [
    (Story) => {
      return (
        <BaseWrapper>
          <Story />
        </BaseWrapper>
      );
    },
  ],
  component: Disable, // BillableStepUsage,
  tags: ['autodocs'],
  title: 'BillableStepUsage',
};

export default meta;
// type Story = StoryObj<typeof BillableStepUsage>;

export const Primary = {};
