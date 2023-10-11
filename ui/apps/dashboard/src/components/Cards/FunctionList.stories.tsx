import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '@/app/layout';
import { FunctionList, ulClassname } from './FunctionList';

const meta: Meta<typeof FunctionList> = {
  args: {
    functions: [
      { name: 'Handle failed payments', slug: 'app-handle-failed-payments' },
      { name: 'Send billing receipt', slug: 'app-send-billing-receipt' },
      {
        name: 'Send discount offer for user feedback',
        slug: 'app-send-discount-offer-for-user-feedback',
      },
    ],
    baseHref: '/env/prod/functions',
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
  component: FunctionList,
  tags: ['autodocs'],
  title: 'FunctionList',
};

export default meta;
type Story = StoryObj<typeof FunctionList>;

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

export const Empty: Story = {
  args: {
    functions: [],
  },
};
