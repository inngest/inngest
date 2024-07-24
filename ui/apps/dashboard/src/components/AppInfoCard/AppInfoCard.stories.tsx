import type { PropsWithChildren } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import { EnvironmentContext } from '@/components/Environments/environment-context';
import { frameworks } from '@/components/FrameworkInfo';
import { languages } from '@/components/LanguageInfo';
import { platforms } from '@/components/PlatformInfo';
import { EnvironmentType, type Environment } from '@/utils/environments';
import { AppInfoCard } from './AppInfoCard';

type PropsAndCustomArgs = React.ComponentProps<typeof AppInfoCard> & {
  framework: string;
  language: string;
  platform: string;
};

const now = new Date();

const meta = {
  args: {
    framework: frameworks[0],
    language: languages[0],
    platform: platforms[0],
  },
  argTypes: {
    framework: {
      options: [...frameworks, 'unknown'],
      control: { type: 'select' },
    },
    language: {
      options: [...languages, 'unknown'],
      control: { type: 'select' },
    },
    platform: {
      options: [...platforms, 'unknown'],
      control: { type: 'select' },
    },
  },
  component: AppInfoCard,
  parameters: {
    layout: 'centered',
  },
  render: ({ framework, language, platform }) => {
    return (
      <DummyEnvContext>
        <AppInfoCard
          app={{
            name: 'App Name',
            externalID: 'app-id',
          }}
          sync={{
            framework,
            lastSyncedAt: now,
            platform,
            sdkLanguage: language,
            sdkVersion: '1.0.0',
            status: 'success',
            url: 'https://example.com',
            vercelDeploymentID: 'abc123',
            vercelDeploymentURL: 'https://example.com/api/inngest',
            vercelProjectID: 'my-project',
            vercelProjectURL: 'https://vercel.com/my-project',
          }}
        />
      </DummyEnvContext>
    );
  },
  tags: ['autodocs'],
  title: 'Components/AppInfoCard',
} satisfies Meta<PropsAndCustomArgs>;

export default meta;

type Story = StoryObj<typeof AppInfoCard>;

export const Default: Story = {};

// TODO: Move this to a shared place since other stories will likely need it
function DummyEnvContext({ children }: PropsWithChildren) {
  const value: Environment = {
    type: EnvironmentType.Production,
    id: '00000000-00000000-00000000-00000000',
    hasParent: false,
    name: 'Production',
    slug: 'production',
    webhookSigningKey: 'fake-signing-key',
    createdAt: new Date().toISOString(),
    isArchived: false,
    isAutoArchiveEnabled: null,
    lastDeployedAt: null,
  };

  return <EnvironmentContext.Provider value={value}>{children}</EnvironmentContext.Provider>;
}
