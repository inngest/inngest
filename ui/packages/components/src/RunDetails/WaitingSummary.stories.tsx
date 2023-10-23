import type { Meta, StoryObj } from '@storybook/react';

import { WaitingSummary } from './WaitingSummary';

const meta = {
  title: 'Components/WaitingSummary',
  component: WaitingSummary,
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
} satisfies Meta<typeof WaitingSummary>;

export default meta;

type Story = StoryObj<typeof WaitingSummary>;

const second = 1000;
const minute = 60 * second;

const baseWaitNode = {
  attempt: 0,
  groupID: 'a',
  scheduledAt: new Date(),
  status: 'waiting',
  waitForEventConfig: {
    eventName: 'app/MyEvent',
    expression: 'async.data.foo == event.data.foo',
    timeout: new Date(Date.now() + minute),
  },
} as const;

export const OneWait: Story = {
  args: {
    history: {
      a: baseWaitNode,
    },
  },
};

export const TwoWaits: Story = {
  args: {
    history: {
      a: {
        ...baseWaitNode,
      },
      b: {
        ...baseWaitNode,
        groupID: 'b',
      },
    },
  },
};
