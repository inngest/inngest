import type { Meta, StoryObj } from '@storybook/react';

import Tab from './Tab';

const meta = {
  title: 'Components/Tab',
  component: Tab,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    key: 'event',
    label: 'event',
    tabAction: () => {},
  },
} satisfies Meta<typeof Tab>;

export default meta;

type Story = StoryObj<typeof Tab>;

export const Default: Story = {
  args: {
    active: false,
  },
};

export const Active: Story = {
  args: {
    active: true,
  },
};
