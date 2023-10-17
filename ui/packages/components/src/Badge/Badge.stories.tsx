import type { Meta, StoryObj } from '@storybook/react';

import { Badge } from './Badge';

const meta = {
  title: 'Components/Badge',
  component: Badge,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    children: 'Badge',
  },
} satisfies Meta<typeof Badge>;

export default meta;

type Story = StoryObj<typeof Badge>;

export const Outlined: Story = {
  args: {
    kind: 'outlined',
  },
};

export const Error: Story = {
  args: {
    kind: 'error',
  },
};

export const Solid: Story = {
  args: {
    kind: 'solid',
    className: 'text-orange-400 bg-orange-400/10',
  },
};
