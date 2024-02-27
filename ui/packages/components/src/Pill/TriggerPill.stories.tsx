import type { Meta, StoryObj } from '@storybook/react';

import { TriggerPill } from './TriggerPill';

const meta = {
  title: 'Components/TriggerPill',
  component: TriggerPill,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof TriggerPill>;

export default meta;

type Story = StoryObj<typeof TriggerPill>;

export const Event: Story = {
  args: {
    type: 'EVENT',
    value: 'billing/payment.failed',
  },
};

export const Cron: Story = {
  args: {
    type: 'CRON',
    value: '* * * * *',
  },
};
