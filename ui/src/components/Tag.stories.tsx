import type { Meta, StoryObj } from '@storybook/react';

import Tag from './Tag';

const meta = {
  title: 'Components/Tag',
  component: Tag,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  args: {
    children: 'Tag',
  },
} satisfies Meta<typeof Tag>;

export default meta;

type Story = StoryObj<typeof Tag>;

export const Default: Story = {
  args: {
    className: 'text-white',
  },
};

export const WithLink: Story = {
  args: {
    className: 'text-white',
    href: new URL('http://ingest.com'),
  },
};
