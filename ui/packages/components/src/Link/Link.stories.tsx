import type { Meta, StoryObj } from '@storybook/react';

import { Link } from './Link';

const meta = {
  title: 'Components/Link',
  component: Link,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Links take the user to a new location without performing any side actions. We have specific NavbarLink and Tabs components for navigation. Do not confuse the usage of Links with Buttons.',
      },
    },
  },
  argTypes: {
    children: {
      control: false,
    },
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Link>;

export default meta;

type Story = StoryObj<typeof Link>;

export const ArrowOnHover: Story = {
  args: {
    children: <p>This is a link to inside the app</p>,
    arrowOnHover: true,
    href: '/app',
  },
  parameters: {
    docs: {
      description: {
        story: 'Takes users to other pages or sections inside the app. It uses Next.js Links.',
      },
    },
  },
};

export const MediumLink: Story = {
  args: {
    children: <p>This is a link to outside the app</p>,
    size: 'medium',
    href: 'inngest.com',
  },
  parameters: {
    docs: {
      description: {
        story: 'Outside links take users outside the app, in a new tab.',
      },
    },
  },
};
