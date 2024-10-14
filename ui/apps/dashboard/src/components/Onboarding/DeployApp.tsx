import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import CommandBlock from '@inngest/components/CodeBlock/CommandBlock';
import { NewLink } from '@inngest/components/Link';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconCloudflare } from '@inngest/components/icons/platforms/Cloudflare';
import { IconFlyIo } from '@inngest/components/icons/platforms/FlyIo';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiCheckboxCircleFill, RiCloudLine } from '@remixicon/react';

import { Secret } from '@/components/Secret';
import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';

export default function DeployApp() {
  const { updateLastCompletedStep } = useOnboardingStep();
  const router = useRouter();

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        Once your app is created, merge your changes and prepare your environment to deploy. Choose
        your preferred hosting provider below and learn how to configure environment variables for a
        secure production setup.
      </p>

      <h4 className="mb-4 text-sm font-medium">Choosing hosting provider:</h4>
      <TabCards defaultValue="all">
        <TabCards.ButtonList>
          <TabCards.Button className="w-32" value="all">
            <div className="flex items-center gap-1.5">
              <RiCloudLine className="h-5 w-5" /> All providers
            </div>
          </TabCards.Button>
          <TabCards.Button className="w-32" value="vercel">
            <div className="flex items-center gap-1.5">
              <IconVercel className="h-4 w-4" /> Vercel
            </div>
          </TabCards.Button>
          <TabCards.Button className="w-32" value="cloudflare">
            <div className="flex items-center gap-1.5">
              <IconCloudflare className="h-5 w-5" /> Cloudflare
            </div>
          </TabCards.Button>
          <TabCards.Button className="w-32" value="flyio">
            <div className="flex items-center gap-1.5">
              <IconFlyIo className="h-4 w-4" /> Fly.io
            </div>
          </TabCards.Button>
        </TabCards.ButtonList>
        <TabCards.Content value="all">
          <div className="mb-4 flex items-center gap-2">
            <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
              <RiCloudLine className="text-basis h-4 w-4" />
            </div>
            <p className="text-basis">All hosting providers (Docker, Kubernetes, etc.)</p>
          </div>
          <p className="mb-4 text-sm">
            Add the two environment variables required for your app to securely communicate with
            Inngest. The process for adding these variables depends on your system setup.
          </p>
          <p className="mb-6 text-sm">
            These variables are compatible with any platform or runtime, including Docker,
            Kubernetes, and others.{' '}
            <NewLink
              size="small"
              href="https://www.inngest.com/docs/events/creating-an-event-key?ref=app-onboarding-deploy-app"
              className="inline-block"
            >
              Learn more about adding keys
            </NewLink>
          </p>
          <div className="text-basis text-sm font-medium">Event key</div>
          <p className="mb-2 text-sm">
            The{' '}
            <code className="text-basis bg-canvasMuted rounded-sm px-1.5 py-0.5 text-xs">
              INNGEST_EVENT_KEY
            </code>{' '}
            is used for sending events and invoking functions
          </p>
          {/* TODO: get prod event key */}
          <Secret kind="event-key" secret="key" className="mb-4" />
          <div className="text-basis text-sm font-medium">Signing key</div>
          <p className="mb-2 text-sm">
            The{' '}
            <code className="text-basis bg-canvasMuted rounded-sm px-1.5 py-0.5 text-xs">
              INNGEST_SIGNING_KEY
            </code>{' '}
            is used for authenticating requests between Inngest and your app
          </p>
          {/* TODO: get prod signing key */}
          <Secret kind="signing-key" secret="key" className="mb-6" />
          <NewButton
            label="Next"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.DeployApp);
              router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.SyncApp }));
            }}
          />
        </TabCards.Content>
        <TabCards.Content value="vercel">
          <div className="mb-4 flex items-center gap-2">
            <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
              <IconVercel className="text-basis h-4 w-4" />
            </div>
            <p className="text-basis">Vercel</p>
          </div>
          <p className="mb-4 text-sm">
            The Vercel integration enables you to host your Inngest functions on the Vercel platform
            and automatically syncs them every time you deploy code.{' '}
            <NewLink
              size="small"
              href="https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-deploy-app"
              className="inline-block"
            >
              Read our Vercel documentation
            </NewLink>
          </p>
          {/* TODO: wire vercel integration flow */}
          <div className="border-subtle divide-subtle mb-4 divide-y border text-sm">
            <div className="flex items-center gap-2 px-3 py-2">
              <RiCheckboxCircleFill className="text-primary-moderate h-4 w-4" /> Auto-syncs on every
              deploy
            </div>
            <div className="flex items-center gap-2 px-3 py-2">
              <RiCheckboxCircleFill className="text-primary-moderate h-4 w-4" /> Branch environments
            </div>
          </div>
          <NewButton
            label="Next"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.DeployApp);
              router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.SyncApp }));
            }}
          />
        </TabCards.Content>
        <TabCards.Content value="cloudflare">
          <div className="mb-4 flex items-center gap-2">
            <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
              <IconCloudflare className="text-basis h-5 w-5" />
            </div>
            <p className="text-basis">Cloudflare</p>
          </div>
          <p className="mb-4 text-sm">
            You can configure the environment variables on Cloudflare using Wrangler or through
            their dashboard. Learn how to define them{' '}
            <NewLink
              size="small"
              href="https://developers.cloudflare.com/workers/configuration/environment-variables/"
              className="inline-block"
            >
              here
            </NewLink>
          </p>
          <div className="text-basis mb-2 text-sm font-medium">Event key</div>
          {/* TODO: get prod event key */}
          <Secret kind="event-key" secret="key" className="mb-4" />
          <div className="text-basis mb-2 text-sm font-medium">Signing key</div>
          {/* TODO: get prod signing key */}
          <Secret kind="signing-key" secret="key" className="mb-6" />
          <NewButton
            label="Next"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.DeployApp);
              router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.SyncApp }));
            }}
          />
        </TabCards.Content>
        <TabCards.Content value="flyio">
          <div className="mb-4 flex items-center gap-2">
            <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
              <IconFlyIo className="h-5 w-5" />
            </div>
            <p className="text-basis">Fly.io</p>
          </div>
          <p className="mb-4 text-sm">
            You can configure the environment variables on Fly.io by adding the values below. Learn
            more about how to set a secret in Fly.io{' '}
            <NewLink
              size="small"
              href="https://fly.io/docs/rails/the-basics/configuration/"
              className="inline-block"
            >
              here
            </NewLink>
          </p>
          {/* TODO: get prod event key */}
          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between pr-4">
              <CommandBlock.Tabs
                tabs={[
                  {
                    title: 'Event key',
                    content: 'fly secrets set INNGEST_EVENT_KEY=',
                    language: 'shell',
                  },
                ]}
                activeTab="Event key"
              />
              <CommandBlock.CopyButton content="fly secrets set INNGEST_EVENT_KEY=" />
            </CommandBlock.Header>
            <CommandBlock
              currentTabContent={{
                title: 'Event key',
                content: 'fly secrets set INNGEST_EVENT_KEY=',
                language: 'shell',
              }}
            />
          </CommandBlock.Wrapper>
          <div className="mb-4" />
          {/* TODO: get prod signing key */}
          <CommandBlock.Wrapper>
            <CommandBlock.Header className="flex items-center justify-between pr-4">
              <CommandBlock.Tabs
                tabs={[
                  {
                    title: 'Signing key',
                    content: 'fly secrets set INNGEST_SIGNING_KEY=',
                    language: 'shell',
                  },
                ]}
                activeTab="Signing key"
              />
              <CommandBlock.CopyButton content="fly secrets set INNGEST_SIGNING_KEY=" />
            </CommandBlock.Header>
            <CommandBlock
              currentTabContent={{
                title: 'Signing key',
                content: 'fly secrets set INNGEST_SIGNING_KEY=',
                language: 'shell',
              }}
            />
          </CommandBlock.Wrapper>
          <div className="mb-4" />
          <NewButton
            label="Next"
            onClick={() => {
              updateLastCompletedStep(OnboardingSteps.DeployApp);
              router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.SyncApp }));
            }}
          />
        </TabCards.Content>
      </TabCards>
    </div>
  );
}
