import { Button } from '@inngest/components/Button';
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

export const AlertWithButton: Story = {
  args: {
    severity: 'error',
    children: 'This is an error message',
    button: (
      <Button onClick={() => {}} kind="secondary" appearance="outlined" label="Refresh Page" />
    ),
  },
};

export const AlertWithLinkInDescription: Story = {
  args: {
    severity: 'error',
    children: (
      <p>
        This is an error message.{' '}
        <Alert.Link href="" className="inline-flex" severity="error">
          This is a link inline
        </Alert.Link>
      </p>
    ),
  },
};
