import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '../app/layout';
import Button from './Button';

const meta: Meta<typeof Button> = {
  component: Button,
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <Story />
        </RootLayout>
      );
    },
  ],
  tags: ['autodocs'],
  title: 'Button',
};

export default meta;
type Story = StoryObj<typeof Button>;

export const Primary: Story = {
  args: {
    children: 'Click Me!',
    variant: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    children: 'Click Me!',
    variant: 'secondary',
  },
};

export const Text: Story = {
  args: {
    children: 'Click Me!',
    variant: 'text',
  },
};

export const TextDanger: Story = {
  args: {
    children: 'Click Me!',
    variant: 'text-danger',
  },
};
