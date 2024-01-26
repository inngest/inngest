import type { Meta } from '@storybook/react';

import { BaseWrapper } from '@/app/baseWrapper';

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
