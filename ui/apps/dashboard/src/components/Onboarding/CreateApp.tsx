import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { NewLink } from '@inngest/components/Link';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiCheckboxCircleFill, RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';

const tabs = [
  {
    title: 'npm',
    content: 'npm install inngest',
    language: 'shell', // TODO: add shell language to monaco
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

  {
    /* TODO: add request to check dev server with polling */
  }
  const devServerIsRunning = (): boolean => {
    return Math.random() < 0.5;
  };

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
        <div className="flex items-center justify-between gap-2 p-4">
          <div>
            <div className="mb-1 flex items-center gap-1">
              <p className=" text-basis text-base font-medium">Dev Server UI</p>
              {devServerIsRunning() && (
                <div className="text-success flex items-center gap-0.5 text-sm">
                  <RiCheckboxCircleFill className="h-4 w-4" />
                  Running
                </div>
              )}
            </div>
            <p className="text-sm">
              Open the Dev Server at{' '}
              <code className="text-basis bg-canvasMuted rounded-sm px-1.5 py-0.5 text-xs">
                http://localhost:8288
              </code>{' '}
              and follow the guide to create your app.
            </p>
          </div>
          {devServerIsRunning() ? (
            <NewButton
              icon={<RiExternalLinkLine />}
              iconSide="left"
              appearance="outlined"
              label="Open"
              href="http://localhost:8288"
              target="_blank"
              rel="noopener noreferrer"
            />
          ) : (
            <div className="text-link flex items-center gap-1.5 text-sm">
              <IconSpinner className="fill-link h-4 w-4" />
              Searching
            </div>
          )}
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
        {/* TODO: add tracking */}
        <NewButton
          appearance="outlined"
          label="I already have an Inngest app"
          onClick={() => {
            updateLastCompletedStep(OnboardingSteps.CreateApp);
            router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.DeployApp }));
          }}
        />
      </div>
    </div>
  );
}
