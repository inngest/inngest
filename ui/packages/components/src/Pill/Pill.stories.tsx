import { RiArrowRightLine, RiErrorWarningFill } from '@remixicon/react';
import type { Meta, StoryObj } from '@storybook/react';

import { Pill } from './Pill';

const meta = {
  title: 'Components/Pill',
  component: Pill,
  parameters: {
    layout: 'centered',
  },
  args: {
    children: 'Pill',
  },
} satisfies Meta<typeof Pill>;

export default meta;

type Story = StoryObj<typeof Pill>;

export const Default: Story = {};

export const WithLink: Story = {
  args: {
    href: 'https://inngest.com',
  },
};

export const WithAction: Story = {
  args: {
    appearance: 'outlined',
    kind: 'error',
    icon: <RiErrorWarningFill className="h-3.5 w-3.5" />,
    iconSide: 'left',
    children: 'Hobby account execution limit reached. Upgrade your plan to continue using Inngest.',
    action: (
      <button className="flex items-center gap-0.5 underline underline-offset-2">
        Upgrade
        <RiArrowRightLine className="h-3.5 w-3.5" />
      </button>
    ),
  },
};
