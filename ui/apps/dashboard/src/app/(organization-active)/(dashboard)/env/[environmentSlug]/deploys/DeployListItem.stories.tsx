import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '@/app/layout';
import { DeployListItem } from './DeployListItem';

const meta: Meta<typeof DeployListItem> = {
  args: {
    createdAt: new Date().toISOString(),
    activeFunctionCount: 31,
    removedFunctionCount: 29,
    status: 'success',
  },
  component: DeployListItem,
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <ul className="w-96 bg-transparent">
            <Story />
          </ul>
        </RootLayout>
      );
    },
  ],
  tags: ['autodocs'],
  title: 'DeployListItem',
};

export default meta;
type Story = StoryObj<typeof DeployListItem>;

export const Success: Story = {
  args: {
    status: 'success',
  },
};

export const Failed: Story = {
  args: {
    status: 'failed',
  },
};

export const NoFunctionCountDelta: Story = {
  args: {
    removedFunctionCount: meta.args?.activeFunctionCount,
  },
};

export const NegativeFunctionCountDelta: Story = {
  args: {
    removedFunctionCount: (meta.args?.activeFunctionCount ?? 0) + 1,
  },
};

export const MassiveFunctionCount: Story = {
  args: {
    activeFunctionCount: 98765,
  },
};

export const WithError: Story = {
  args: {
    error: 'Oh snap!',
  },
};

export const Selected: Story = {
  args: {
    isSelected: true,
  },
};

export const MissingFunctionCount: Story = {
  args: {
    activeFunctionCount: undefined,
  },
};

export const MissingremovedFunctionCount: Story = {
  args: {
    removedFunctionCount: undefined,
  },
};

const OneMinute = 1000 * 60;
const OneHour = OneMinute * 60;
const OneDay = OneHour * 24;
const OneWeek = OneDay * 7;

export const TwoMinutesAgo: Story = {
  args: {
    createdAt: new Date(Date.now() - OneMinute * 2).toISOString(),
  },
};

export const TwoHoursAgo: Story = {
  args: {
    createdAt: new Date(Date.now() - OneHour * 2).toISOString(),
  },
};

export const TwoDaysAgo: Story = {
  args: {
    createdAt: new Date(Date.now() - OneDay * 2).toISOString(),
  },
};

export const TwoWeeksAgo: Story = {
  args: {
    createdAt: new Date(Date.now() - OneWeek * 2).toISOString(),
  },
};
