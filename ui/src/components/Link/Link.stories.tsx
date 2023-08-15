import type { Meta, StoryObj } from '@storybook/react';

import Link from './Link';

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
    onClick: { action: 'archor clicked'}
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Link>;

export default meta;

type Story = StoryObj<typeof Link>;

export const InternalNavigationUsingNextLink: Story = {
  args: {
    children: <p>This is a link to inside the app</p>,
    internalNavigation: true,
    href: '/app'
  },
  parameters: {
    docs: {
      description: {
        story: 'It is the preferable solution to take users to other pages or sections inside the app. It uses Next.js Links.'
      }
    }
  }
};

export const InternalNavigationUsingAnchor: Story = {
  args: {
    children: <p>This is a link to inside the app</p>,
    internalNavigation: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'It is the solution for internal navigation that requires onClick. Suitable for cases where we use a dispatch to see in-app docs.'
      }
    }
  }
};

export const ExternalNavigation: Story = {
  args: {
    children: <p>This is a link to outside the app</p>,
    internalNavigation: false,
  },
  parameters: {
    docs: {
      description: {
        story: 'Outside links take users outside the app, in a new tab.'
      }
    }
  }
};
