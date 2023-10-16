import { IconStatusCircleCheck } from '@inngest/components/icons/StatusCircleCheck';
import type { Meta, StoryObj } from '@storybook/react';

import { TimelineNodeHeader } from './TimelineNodeHeader';

const meta = {
  title: 'Components/TimelineNodeHeader',
  component: TimelineNodeHeader,
  parameters: {
    layout: 'centered',
  },
  decorators: [
    (Story) => {
      return (
        <div style={{ width: 600 }}>
          <Story />
        </div>
      );
    },
  ],
  argTypes: {
    icon: {
      options: [<IconStatusCircleCheck />],
      control: { type: 'select' },
    },
  },
  tags: ['autodocs'],
} satisfies Meta<typeof TimelineNodeHeader>;

export default meta;

type Story = StoryObj<typeof TimelineNodeHeader>;

export const Default: Story = {
  args: {
    icon: <IconStatusCircleCheck />,
    badge: 'Step',
    title: 'This is a Function Step',
    metadata: {
      label: 'Queued At:',
      value: '24/09/2023, 11:48:03',
    },
  },
};

export const NoBadge: Story = {
  args: {
    icon: <IconStatusCircleCheck />,
    title: 'This is a Function',
    metadata: {
      label: 'Queued At:',
      value: '24/09/2023, 11:48:03',
    },
  },
};
