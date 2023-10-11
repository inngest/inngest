import type { Meta, StoryObj } from '@storybook/react';

import RootLayout from '@/app/layout';
import DeployCard from './DeployCard';

const meta: Meta<typeof DeployCard> = {
  args: {
    id: 'deadbeef-cafe-0000-0000-000000000000',
    appName: 'My App',
    checksum: '42b5731a6336d7a937d1c68c71e11a5e8a2deccb',
    createdAt: new Date().toISOString(),
    deployedFunctions: [
      { slug: 'app-send-upgrade-email', name: 'Send upgrade email' },
      { slug: 'app-handle-failed-payments', name: 'Handle failed payments' },
    ],
    environmentSlug: 'my-branch',
    removedFunctions: [{ slug: 'app-send-welcome-email', name: 'Send welcome email' }],
    sdkLanguage: 'node',
    sdkVersion: '1.2.3',
    status: 'success',
  },
  decorators: [
    (Story) => {
      return (
        <RootLayout>
          <Story />
        </RootLayout>
      );
    },
  ],
  component: DeployCard,
  tags: ['autodocs'],
  title: 'DeployCard',
};

export default meta;
type Story = StoryObj<typeof DeployCard>;

export const Primary: Story = {};
