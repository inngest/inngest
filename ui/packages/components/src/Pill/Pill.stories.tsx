import type { Meta, StoryObj } from '@storybook/react';

import { Pill } from './Pill';

const meta = {
  title: 'Components/Pill',
  component: Pill,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    children: 'Pill',
  },
} satisfies Meta<typeof Pill>;

export default meta;

type Story = StoryObj<typeof Pill>;

export const Default: Story = {};

export const WithLink: Story = {
  args: {
    href: new URL('http://inngest.com'),
  },
};
