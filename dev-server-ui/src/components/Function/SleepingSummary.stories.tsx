import type { Meta, StoryObj } from '@storybook/react';

import { SleepingSummary } from './SleepingSummary';

const meta = {
  title: 'Components/SleepingSummary',
  component: SleepingSummary,
  decorators: [
    (Story) => {
      return (
        <div style={{ width: 600 }}>
          <Story />
        </div>
      );
    },
  ],
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof SleepingSummary>;

export default meta;

type Story = StoryObj<typeof SleepingSummary>;

const second = 1000;
const minute = 60 * second;

const baseSleepNode = {
  attempt: 0,
  groupID: 'a',
  scheduledAt: new Date(),
  sleepConfig: {
    until: new Date(Date.now() + minute),
  },
  status: 'sleeping',
} as const;

export const OneSleep: Story = {
  args: {
    history: {
      a: baseSleepNode,
    },
  },
};

export const TwoSleeps: Story = {
  args: {
    history: {
      a: {
        ...baseSleepNode,
      },
      b: {
        ...baseSleepNode,
        groupID: 'b',
      },
    },
  },
};
