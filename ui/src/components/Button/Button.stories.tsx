import type { Meta, StoryObj } from '@storybook/react';

import { IconChevron } from '@/icons';
import Button from './Button';

const meta = {
  title: 'Components/NewButton',
  component: Button,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    label: 'Click me',
  },
  argTypes: {
    keys: {
      options: [[], ['↵'], ['⌘', 'A']],
      control: { type: 'select' },
    },
  },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof Button>;

export const Primary: Story = {
  args: {
    kind: 'primary',
  },
};

export const Default: Story = {
  args: {
    kind: 'default',
  },
};

export const Solid: Story = {
  args: {
    appearance: 'solid',
  },
};

export const Outlined: Story = {
  args: {
    appearance: 'outlined',
  },
};

export const WithIcon: Story = {
  args: {
    kind: 'primary',
    icon: <IconChevron />,
  },
};

export const WithKey: Story = {
  args: {
    kind: 'primary',
    keys: ['⌘', '↵'],
  },
};
