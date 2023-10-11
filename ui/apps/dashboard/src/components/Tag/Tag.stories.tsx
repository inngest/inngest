import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '../../app/layout';
import { Tag } from './Tag';

const meta: Meta<typeof Tag> = {
  component: Tag,
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <Story />
        </RootLayout>
      );
    },
  ],
  tags: ['autodocs'],
  title: 'Tag',
};

export default meta;
type Story = StoryObj<typeof Tag>;

export const Solid: Story = {
  args: {
    children: 'This is a tag',
    kind: 'solid',
  },
};

export const Subtle: Story = {
  args: {
    children: 'This is a tag',
    kind: 'subtle',
  },
};

export const Clickable: Story = {
  args: {
    children: 'Click Me!',
    href: new URL('http://www.inngest.com'),
  },
};
