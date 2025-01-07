import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { InlineCode } from '@inngest/components/Code';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { Link } from '@inngest/components/Link';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { useDevServer } from '@inngest/components/utils/useDevServer';
import { RiCheckboxCircleFill, RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';
import { useOnboardingTracking } from './useOnboardingTracking';
import { getNextStepName } from './utils';

const tabs = [
  {
    title: 'npm',
    content: 'npx inngest-cli@latest dev',
    language: 'shell',
  },
  {
    title: 'yarn',
    content: 'yarn dlx inngest-cli@latest dev',
    language: 'shell',
  },
  {
    title: 'pnpm',
    content: 'pnpm dlx inngest-cli@latest dev',
    language: 'shell',
  },
];

export default function CreateApp() {
  const currentStepName = OnboardingSteps.CreateApp;
  const nextStepName = getNextStepName(currentStepName);
  const { updateCompletedSteps } = useOnboardingStep();
  const [activeTab, setActiveTab] = useState(tabs[0]?.title || '');
  const currentTabContent = tabs.find((tab) => tab.title === activeTab) || tabs[0];
  const router = useRouter();
  const { isRunning: devServerIsRunning } = useDevServer(2500);
  const tracking = useOnboardingTracking();

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        An Inngest &quot;App&quot; is a group of functions served on a single endpoint or server.
        The first step is to create your app and functions, serve it, and test it locally with the
        Inngest Dev Server.
      </p>
      <p className="mb-6 text-sm">
        The Dev Server will guide you through setup and help you build and test functions end to
        end.{' '}
        <Link
          className="inline-block"
          size="small"
          target="_blank"
          href="https://www.inngest.com/docs/local-development?ref=app-onboarding-create-app"
        >
          Learn more about local development
        </Link>
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
              <p className=" text-basis text-base font-medium">Dev Server</p>
              {devServerIsRunning && (
                <div className="text-success flex items-center gap-0.5 text-sm">
                  <RiCheckboxCircleFill className="h-4 w-4" />
                  Running
                </div>
              )}
            </div>
            <p className="text-sm">
              Open the Dev Server at <InlineCode>http://localhost:8288</InlineCode> and follow the
              guide to create your app.
            </p>
          </div>
          {devServerIsRunning ? (
            <Button
              icon={<RiExternalLinkLine />}
              iconSide="left"
              appearance="outlined"
              label="Open"
              href="http://localhost:8288"
              target="_blank"
              rel="noopener noreferrer"
              onClick={() =>
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: { type: 'btn-click', label: 'open-dev-server' },
                })
              }
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
        <Button
          label="Next"
          disabled={!devServerIsRunning}
          onClick={() => {
            updateCompletedSteps(currentStepName, {
              metadata: {
                completionSource: 'manual',
              },
            });
            tracking?.trackOnboardingAction(currentStepName, {
              metadata: { type: 'btn-click', label: 'next' },
            });
            router.push(pathCreator.onboardingSteps({ step: nextStepName }));
          }}
        />
        <Button
          appearance="outlined"
          label="I already have an Inngest app"
          onClick={() => {
            updateCompletedSteps(currentStepName, {
              metadata: {
                completionSource: 'manual',
              },
            });
            tracking?.trackOnboardingAction(currentStepName, {
              metadata: { type: 'btn-click', label: 'skip' },
            });
            router.push(pathCreator.onboardingSteps({ step: nextStepName }));
          }}
        />
      </div>
    </div>
  );
}
