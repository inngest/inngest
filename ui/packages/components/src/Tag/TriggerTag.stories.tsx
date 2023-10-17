import { TriggerTypes } from '@inngest/components/types/triggers';
import type { Meta, StoryObj } from '@storybook/react';

import { TriggerTag } from './TriggerTag';

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
    type: TriggerTypes.Event,
    value: 'billing/payment.failed',
  },
};

export const Cron: Story = {
  args: {
    type: TriggerTypes.Cron,
    value: '* * * * *',
  },
};
