import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { NewLink } from '@inngest/components/Link';

import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';

const tabs = [
  {
    title: 'npm',
    content: 'npm install inngest',
    language: 'shell',
  },
  {
    title: 'yarn',
    content: 'yarn add inngest',
    language: 'shell',
  },
  {
    title: 'pnpm',
    content: 'pnpm add inngest',
    language: 'shell',
  },
  {
    title: 'bun',
    content: 'bun add inngest',
    language: 'shell',
  },
];

export default function CreateApp() {
  const { updateLastCompletedStep } = useOnboardingStep();
  const [activeTab, setActiveTab] = useState(tabs[0]?.title || '');
  const currentTabContent = tabs.find((tab) => tab.title === activeTab) || tabs[0];
  const router = useRouter();

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        Inngest “App” is a group of functions served on a single endpoint or server. The first step
        is to create your app and functions, serve it, and test it locally with the Inngest Dev
        Server.
      </p>
      <p className="mb-6 text-sm">
        The Dev Server will guide you through setup and help you build and test functions end to
        end.{' '}
        <NewLink
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/local-development?ref=app-onboarding-create-app"
        >
          Learn more about local development
        </NewLink>
      </p>
      <p className="mb-2 text-sm">
        Run the following CLI command on your machine to get the Inngest Dev Server started locally:
      </p>
      <CommandBlock.Wrapper>
        <CommandBlock.Header className="flex items-center justify-between pr-4">
          <CommandBlock.Tabs tabs={tabs} activeTab={activeTab} setActiveTab={setActiveTab} />
          <CommandBlock.CopyButton content={currentTabContent?.content} />
        </CommandBlock.Header>
        <CommandBlock currentTabContent={currentTabContent} />
      </CommandBlock.Wrapper>
      <Card className="my-6">
        <div className="p-4">
          <p className=" text-basis text-base font-medium">Dev Server UI</p>
          <p className="text-sm">
            Open the Dev Server at{' '}
            <code className="text-basis bg-canvasMuted rounded-sm px-1.5 py-0.5 text-xs">
              http://localhost:8288
            </code>{' '}
            and follow the guide to create your app.
          </p>
        </div>
      </Card>
      <div className="flex items-center gap-2">
        <NewButton
          label="Next"
          onClick={() => {
            updateLastCompletedStep(OnboardingSteps.CreateApp);
            router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.DeployApp }));
          }}
        />
        <NewButton appearance="outlined" label="I already have an Inngest app" />
      </div>
    </div>
  );
}
