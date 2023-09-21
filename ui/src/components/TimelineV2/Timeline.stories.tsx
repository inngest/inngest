import type { Meta, StoryObj } from '@storybook/react';

import type { RunHistoryItem } from '@/store/generated';
import { HistoryParser } from './historyParser/historyParser';
import { Timeline } from './index';
import succeedsWith2StepsData from './storyData/succeedsWith2Steps.json';
import waitsForEventData from './storyData/waitsForEvent.json';

const meta = {
  title: 'Components/Timeline',
  component: Timeline,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Timeline>;

export default meta;

type Story = StoryObj<typeof Timeline>;

export const succeedsWith2Steps: Story = {
  args: {
    history: new HistoryParser(succeedsWith2StepsData as RunHistoryItem[]).history,
  },
};

export const waitsForEvent: Story = {
  args: {
    history: new HistoryParser(waitsForEventData as RunHistoryItem[]).history,
  },
};
