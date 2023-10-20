import type { Meta, StoryObj } from '@storybook/react';

import Header from './Header';

const meta = {
  title: 'Layout/Header',
  component: Header,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Header>;

export default meta;

type Story = StoryObj<typeof Header>;

export const Default: Story = {
  args: {
    children: <p className="pr-4 text-white">This is the header's middle component</p>,
  },
};
