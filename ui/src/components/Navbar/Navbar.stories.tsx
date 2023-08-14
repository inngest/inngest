import type { Meta, StoryObj } from '@storybook/react';

import Navbar from './Navbar';
import NavbarLink from './NavbarLink';
import { Default, WithCounter } from './NavbarLink.stories';

const meta = {
  title: 'Layout/Navbar',
  component: Navbar,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    children: {
      control: false,
    },
  },
} satisfies Meta<typeof Navbar>;

export default meta;

type Story = StoryObj<typeof Navbar>;

export const NavbarWithLinks: Story = {
  render: (args) => (
    <Navbar {...args}>
      <NavbarLink {...Default.args} />
      <NavbarLink {...WithCounter.args} />
    </Navbar>
  ),
};
