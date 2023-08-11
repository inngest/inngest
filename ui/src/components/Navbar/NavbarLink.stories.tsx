import type { Meta, StoryObj } from '@storybook/react';

import { IconWindow } from '@/icons';
import NavbarLink from './NavbarLink';

const meta = {
  title: 'Layout/NavbarLink',
  component: NavbarLink,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    icon: {
      control: false,
    },
  },
  args: {
    icon: <IconWindow className="h-[1.125rem] w-[1.125rem]" />,
    tabName: 'Apps',
    href: '/',
  },
} satisfies Meta<typeof NavbarLink>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithCounter: Story = {
  args: {
    badge: 0,
  },
};

export const WithError: Story = {
  args: {
    badge: 0,
    hasError: true,
  },
};
