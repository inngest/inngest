import type { Meta, StoryObj } from '@storybook/react';

import SearchInput from './SearchInput';

const meta = {
  title: 'Components/SearchInput',
  component: SearchInput,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    placeholder: 'Search...',
    value: 'Name 1',
  },
} satisfies Meta<typeof SearchInput>;

export default meta;

type Story = StoryObj<typeof SearchInput>;

export const Default: Story = {};

export const Empty: Story = {
  args: {
    value: '',
  },
};
