import type { Meta, StoryObj } from '@storybook/react';

import { Alert } from '.';

const meta = {
  title: 'Components/Alert',
  component: Alert,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Alert>;

export default meta;

type Story = StoryObj<typeof Alert>;

export const Error: Story = {
  args: {
    severity: 'error',
    children: 'This is an error message',
  },
};

export const Warning: Story = {
  args: {
    severity: 'warning',
    children: 'This is a warning message',
  },
};

export const Info: Story = {
  args: {
    severity: 'info',
    children: 'This is an info message',
  },
};

export const Success: Story = {
  args: {
    severity: 'success',
    children: 'This is a success message',
  },
};
