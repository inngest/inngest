import type { Meta, StoryObj } from '@storybook/react';

import { IconFunction, IconWindow } from '@/icons';
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
} satisfies Meta<typeof NavbarLink>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    icon: <IconFunction />,
    tabName: 'Functions',
    href: '/function',
  },
};

export const WithCounter: Story = {
  args: {
    badge: 0,
    icon: <IconWindow className="h-[1.125rem] w-[1.125rem]" />,
    tabName: 'Apps',
    href: '/apps',
  },
};

export const WithError: Story = {
  args: {
    badge: 0,
    hasError: true,
    icon: <IconWindow className="h-[1.125rem] w-[1.125rem]" />,
    tabName: 'Apps',
    href: '/apps',
  },
};
