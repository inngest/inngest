import type { Meta, StoryObj } from '@storybook/react';

import { HistoryParser, type RawHistoryItem } from '../utils/historyParser';
import { CancellationSummary } from './CancellationSummary';

const meta = {
  title: 'Components/CancellationSummary',
  component: CancellationSummary,
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
} satisfies Meta<typeof CancellationSummary>;

export default meta;

type Story = StoryObj<typeof CancellationSummary>;

const baseItem: RawHistoryItem = {
  attempt: 0,
  cancel: {
    eventID: '01HAWKZ4BD4MHGCXSWTXHTJVGN',
    expression: null,
    userID: null,
  },
  createdAt: '2023-09-21T15:37:45.720719-04:00',
  functionVersion: 1,
  groupID: 'ebf329b0-badc-4cc9-b962-514f5202f2fc',
  id: '01HAWKZ4FRPHGNVMPBJ8DBCCR5',
  sleep: null,
  stepName: null,
  type: 'FunctionCancelled',
  url: null,
  waitForEvent: null,
  waitResult: null,
};

export const Main: Story = {
  args: {
    history: new HistoryParser([baseItem]),
  },
};
