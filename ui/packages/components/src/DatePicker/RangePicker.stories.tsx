import type { Meta, StoryObj } from '@storybook/react';

import { RangePicker } from './RangePicker';

const meta = {
  title: 'Components/RangePicker',
  component: RangePicker,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {},
} satisfies Meta<typeof RangePicker>;

export default meta;

type Story = StoryObj<typeof RangePicker>;

export const DefaultRangePicker: Story = {
  args: {},
};
