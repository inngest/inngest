import type { Meta, StoryObj } from '@storybook/react';

import { TimeRangeInput } from './index';

const meta = {
  title: 'Components/TimeRangeInput',
  component: TimeRangeInput,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof TimeRangeInput>;

export default meta;

type Story = StoryObj<typeof TimeRangeInput>;

export const Main: Story = {};
