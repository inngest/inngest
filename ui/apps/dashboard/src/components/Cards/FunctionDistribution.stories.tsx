import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '@/app/layout';
import { FunctionDistribution } from './FunctionDistribution';

const meta: Meta<typeof FunctionDistribution> = {
  args: {
    activeCount: 8,
    disabledCount: 4,
    removedCount: 2,
  },
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <Story />
        </RootLayout>
      );
    },
  ],
  component: FunctionDistribution,
  tags: ['autodocs'],
  title: 'FunctionDistribution',
};

export default meta;
type Story = StoryObj<typeof FunctionDistribution>;

export const Primary: Story = {};

export const ManyActive: Story = {
  args: {
    activeCount: 1000,
    disabledCount: 1,
    removedCount: 1,
  },
};

export const ManyDisabled: Story = {
  args: {
    activeCount: 1,
    disabledCount: 1000,
    removedCount: 1,
  },
};

export const ManyRemoved: Story = {
  args: {
    activeCount: 1,
    disabledCount: 1,
    removedCount: 1000,
  },
};

export const NoActive: Story = {
  args: {
    activeCount: 0,
    disabledCount: 1,
    removedCount: 1,
  },
};

export const NoDisabled: Story = {
  args: {
    activeCount: 1,
    disabledCount: 0,
    removedCount: 1,
  },
};

export const NoRemoved: Story = {
  args: {
    activeCount: 1,
    disabledCount: 1,
    removedCount: 0,
  },
};

export const None: Story = {
  args: {
    activeCount: 0,
    disabledCount: 0,
    removedCount: 0,
  },
};

export const OnlyActive: Story = {
  args: {
    activeCount: 1,
    disabledCount: 0,
    removedCount: 0,
  },
};

export const OnlyDisabled: Story = {
  args: {
    activeCount: 0,
    disabledCount: 1,
    removedCount: 0,
  },
};

export const OnlyRemoved: Story = {
  args: {
    activeCount: 0,
    disabledCount: 0,
    removedCount: 1,
  },
};
