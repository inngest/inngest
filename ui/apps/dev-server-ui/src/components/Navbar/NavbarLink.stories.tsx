import { IconApp } from '@inngest/components/icons/App';
import { IconFunction } from '@inngest/components/icons/Function';
import type { Meta, StoryObj } from '@storybook/react';

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

type Story = StoryObj<typeof NavbarLink>;

export const Default: Story = {
  args: {
    icon: <IconFunction />,
    tabName: 'Functions',
    href: '/function',
  },
};

export const WithCounter: Story = {
  args: {
    icon: <IconApp />,
    tabName: 'Apps',
    href: '/apps',
  },
};

export const WithError: Story = {
  args: {
    hasError: true,
    icon: <IconFunction />,
    tabName: 'Apps',
    href: '/apps',
  },
};
