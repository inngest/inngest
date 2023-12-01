import type { Meta, StoryObj } from '@storybook/react';

import { HistoryParser, type RawHistoryItem } from '../utils/historyParser';
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

const baseItem: RawHistoryItem = {
  attempt: 0,
  cancel: null,
  createdAt: '2023-09-22T16:22:38.136906-04:00',
  functionVersion: 1,
  groupID: 'bd178be1-d9ab-42be-9669-15a78eaf9f2a',
  id: '01HAZ8Y0SRB55DT2AX1FAX5DW2',
  sleep: {
    until: '2023-09-22T16:22:48.136637-04:00',
  },
  type: 'StepSleeping',
  stepName: '10s',
  url: null,
  waitForEvent: null,
  waitResult: null,
};

export const OneSleep: Story = {
  args: {
    history: new HistoryParser([baseItem]),
  },
};

export const TwoSleeps: Story = {
  args: {
    history: new HistoryParser([baseItem, { ...baseItem, groupID: 'b' }]),
  },
};
