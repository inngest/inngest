import type { Meta, StoryObj } from '@storybook/react';

import { HistoryParser, type RawHistoryItem } from '../utils/historyParser';
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

const baseItem: RawHistoryItem = {
  attempt: 0,
  cancel: null,
  createdAt: '2023-09-29T11:56:58.808606-04:00',
  functionVersion: 1,
  groupID: 'dca64663-efce-458b-8eff-5fa8e06b11a4',
  id: '01HBGTGM1R3YX0PWD92WXFPZVK',
  sleep: null,
  stepName: 'bar',
  type: 'StepWaiting',
  url: null,
  waitForEvent: {
    eventName: 'bar',
    expression: null,
    timeout: '2023-09-29T11:57:58.808601-04:00',
  },
  waitResult: null,
};

export const OneWait: Story = {
  args: {
    history: new HistoryParser([baseItem]),
  },
};

export const TwoWaits: Story = {
  args: {
    history: new HistoryParser([baseItem, { ...baseItem, groupID: 'b' }]),
  },
};
