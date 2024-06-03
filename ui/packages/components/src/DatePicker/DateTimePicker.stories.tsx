import type { Meta, StoryObj } from '@storybook/react';

import { DateTimePicker } from './DateTimePicker';

const meta = {
  title: 'Components/DateTimePicker',
  component: DateTimePicker,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {},
} satisfies Meta<typeof DateTimePicker>;

export default meta;

type Story = StoryObj<typeof DateTimePicker>;

export const DefaultDateTimePicker: Story = {
  args: { onChange: (d) => console.log(d), defaultValue: new Date() },
};
