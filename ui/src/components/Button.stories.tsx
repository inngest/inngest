import type { Meta, StoryObj } from '@storybook/react';

import { IconChevron } from '@/icons';
import Button from './Button';

const meta = {
  title: 'Components/Button',
  component: Button,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    label: 'Click me',
  },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    kind: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    kind: 'secondary',
  },
};

export const Text: Story = {
  args: {
    kind: 'text',
  },
};

export const WithIcon: Story = {
  args: {
    kind: 'primary',
    icon: <IconChevron />,
  },
};
