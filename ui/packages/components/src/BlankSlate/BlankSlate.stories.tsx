import type { Meta, StoryObj } from '@storybook/react';

import { BlankSlate } from './BlankSlate';

const meta = {
  title: 'Components/BlankSlate',
  component: BlankSlate,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    imageUrl: '/images/no-results.png',
    title: 'This is a title',
    subtitle: 'This is a subtitle',
  },
} satisfies Meta<typeof BlankSlate>;

export default meta;

type Story = StoryObj<typeof BlankSlate>;

export const Default: Story = {};

export const WithLink: Story = {
  args: {
    link: {
      text: 'This is a link',
      url: '/',
    },
  },
};

export const WithButton: Story = {
  args: {
    button: {
      text: 'Click Me',
      onClick: () => {},
    },
  },
};
