import type { Meta, StoryObj } from '@storybook/react';

import { frameworks } from '@/components/FrameworkInfo';
import { languages } from '@/components/LanguageInfo';
import { platforms } from '@/components/PlatformInfo';
import { AppCard } from './AppCard';

type PropsAndCustomArgs = React.ComponentProps<typeof AppCard> & {
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
  component: AppCard,
  parameters: {
    layout: 'centered',
  },
  render: ({ framework, language, platform }) => {
    return (
      <AppCard
        app={{
          name: 'App Name',
          externalID: 'app-id',
          functionCount: 1,
          latestSync: {
            error: null,
            framework,
            lastSyncedAt: now,
            platform,
            sdkLanguage: language,
            sdkVersion: '1.0.0',
            status: 'success',
            url: 'https://example.com',
          },
        }}
        envSlug="fake"
      />
    );
  },
  tags: ['autodocs'],
  title: 'Components/AppCard',
} satisfies Meta<PropsAndCustomArgs>;

export default meta;

type Story = StoryObj<typeof AppCard>;

export const Default: Story = {};
