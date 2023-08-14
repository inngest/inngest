import type { Meta, StoryObj } from '@storybook/react';

import { FunctionTriggerTypes } from '@/store/generated';
import TriggerTag from './TriggerTag';

const meta = {
  title: 'Components/TriggerTag',
  component: TriggerTag,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof TriggerTag>;

export default meta;

type Story = StoryObj<typeof TriggerTag>;

export const Event: Story = {
  args: {
    type: FunctionTriggerTypes.Event,
    value: 'billing/payment.failed',
  },
};

export const Cron: Story = {
  args: {
    type: FunctionTriggerTypes.Cron,
    value: '* * * * *',
  },
};
