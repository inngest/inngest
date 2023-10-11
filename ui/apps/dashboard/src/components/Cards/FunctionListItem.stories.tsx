import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '@/app/layout';
import { ulClassname } from './FunctionList';
import { FunctionListItem } from './FunctionListItem';

const meta: Meta<typeof FunctionListItem> = {
  args: {
    name: 'Handle failed payments',
    href: '/env/prod/functions/function-slug',
    status: 'active',
  },
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <ul className={ulClassname}>
            <Story />
          </ul>
        </RootLayout>
      );
    },
  ],
  component: FunctionListItem,
  tags: ['autodocs'],
  title: 'FunctionListItem',
};

export default meta;
type Story = StoryObj<typeof FunctionListItem>;

export const Active: Story = {
  args: {
    status: 'active',
  },
};

export const Removed: Story = {
  args: {
    status: 'removed',
  },
};
