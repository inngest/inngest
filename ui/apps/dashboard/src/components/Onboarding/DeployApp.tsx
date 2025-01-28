import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code/InlineCode';
import { Link } from '@inngest/components/Link';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { IconCloudflare } from '@inngest/components/icons/platforms/Cloudflare';
import { IconFlyIo } from '@inngest/components/icons/platforms/FlyIo';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { RiCheckboxCircleFill, RiCloudLine } from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { useVercelIntegration } from '@/app/(organization-active)/(dashboard)/settings/integrations/vercel/useVercelIntegration';
import { Secret } from '@/components/Secret';
import { useDefaultEventKey } from '@/queries/useDefaultEventKey';
import { pathCreator } from '@/utils/urls';
import { useEnvironment } from '../Environments/environment-context';
import { OnboardingSteps } from '../Onboarding/types';
import useOnboardingStep from './useOnboardingStep';
import { useOnboardingTracking } from './useOnboardingTracking';
import { getNextStepName } from './utils';

export default function DeployApp() {
  const currentStepName = OnboardingSteps.DeployApp;
  const nextStepName = getNextStepName(currentStepName);
  const { updateCompletedSteps } = useOnboardingStep();
  const router = useRouter();
  const env = useEnvironment();
  const res = useDefaultEventKey({ envID: env.id });
  const defaultEventKey = res.data?.defaultKey.presharedKey || 'Unknown key';
  const tracking = useOnboardingTracking();
  const [, setInstallingVercelFromOnboarding] = useLocalStorage(
    'installingVercelFromOnboarding',
    false
  );

  const { data, isLoading, error } = useVercelIntegration();

  const hasVercelIntegration = Boolean(data);
  const vercelProjects = data?.projects;
  const enabledProjects = vercelProjects?.filter((project) => project.isEnabled) ?? [];

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        Once your app is created, merge your changes and prepare your environment to deploy. Choose
        your preferred hosting provider below and learn how to configure environment variables for a
        secure production setup.
      </p>

      <h4 className="mb-4 text-sm font-medium">Choosing hosting provider:</h4>
      <TabCards defaultValue="vercel">
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
            <p className="text-basis">All hosting providers (AWS, GCP, Azure, etc.)</p>
          </div>
          <p className="mb-4 text-sm">
            Add the two environment variables required for your app to securely communicate with
            Inngest. The process for adding these variables depends on your system setup.
          </p>
          <p className="mb-6 text-sm">
            These variables are compatible with any platform or runtime, including Docker,
            Kubernetes, and others.{' '}
            <Link
              size="small"
              href="https://www.inngest.com/docs/events/creating-an-event-key?ref=app-onboarding-deploy-app"
              className="inline-block"
              target="_blank"
            >
              Learn more about adding keys
            </Link>
          </p>
          <div className="text-basis text-sm font-medium">Event key</div>
          <p className="mb-2 text-sm">
            The <InlineCode>INNGEST_EVENT_KEY</InlineCode> is used for sending events and invoking
            functions
          </p>
          <Secret kind="event-key" secret={defaultEventKey} className="mb-4" />
          <div className="text-basis text-sm font-medium">Signing key</div>
          <p className="mb-2 text-sm">
            The <InlineCode>INNGEST_SIGNING_KEY</InlineCode> is used for authenticating requests
            between Inngest and your app
          </p>
          <Secret kind="signing-key" secret={env.webhookSigningKey} className="mb-6" />
          <Button
            label="Next"
            onClick={() => {
              updateCompletedSteps(currentStepName, {
                metadata: {
                  completionSource: 'manual',
                  hostingProvider: 'all',
                },
              });
              tracking?.trackOnboardingAction(currentStepName, {
                metadata: { type: 'btn-click', label: 'skip', hostingProvider: 'all' },
              });
              tracking?.trackOnboardingAction(currentStepName, {
                metadata: { type: 'btn-click', label: 'next', hostingProvider: 'all' },
              });
              router.push(pathCreator.onboardingSteps({ step: nextStepName }) + '?nonVercel=true');
            }}
          />
        </TabCards.Content>
        <TabCards.Content value="vercel">
          <div className="mb-4 flex items-center justify-between ">
            <div className="flex items-center gap-2">
              <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
                <IconVercel className="text-basis h-4 w-4" />
              </div>
              <p className="text-basis">Vercel</p>
            </div>
            <Button
              label="Manage Vercel integration"
              kind="secondary"
              appearance="outlined"
              onClick={() => {
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: {
                    type: 'btn-click',
                    label: 'view-integration',
                    hostingProvider: 'vercel',
                  },
                });
                router.push(pathCreator.vercel());
              }}
            />
          </div>
          <p className="mb-4 text-sm">
            The Vercel integration enables you to host your Inngest functions on the Vercel platform
            and automatically syncs them every time you deploy code.{' '}
            <Link
              size="small"
              href="https://www.inngest.com/docs/deploy/vercel?ref=app-onboarding-deploy-app"
              className="inline-block"
              target="_blank"
            >
              Read our Vercel documentation
            </Link>
          </p>
          <div className="border-subtle divide-subtle mb-4 divide-y border text-sm">
            <div className="flex items-center gap-2 px-3 py-2">
              <RiCheckboxCircleFill className="text-primary-moderate h-4 w-4" /> Auto-syncs on every
              deploy
            </div>
            <div className="flex items-center gap-2 px-3 py-2">
              <RiCheckboxCircleFill className="text-primary-moderate h-4 w-4" /> Branch environments
            </div>
          </div>
          {!hasVercelIntegration && (
            <div className="flex items-center justify-between">
              <Button
                label="Connect Inngest to Vercel"
                onClick={() => {
                  tracking?.trackOnboardingAction(currentStepName, {
                    metadata: { type: 'btn-click', label: 'connect', hostingProvider: 'vercel' },
                  });
                  setInstallingVercelFromOnboarding(true);
                  router.push(`https://vercel.com/integrations/inngest/new`);
                }}
                disabled={isLoading}
              />
              {isLoading && (
                <div className="text-link flex items-center gap-1.5 text-sm">
                  <IconSpinner className="fill-link h-4 w-4" />
                  Searching for integration
                </div>
              )}
            </div>
          )}
          {hasVercelIntegration && (
            <p className="text-success my-4 text-sm">
              {enabledProjects.length} project{enabledProjects.length === 1 ? '' : 's'} enabled
              successfully
            </p>
          )}
          {error && (
            <Alert className="my-4 text-sm" severity="error">
              {error.message}
            </Alert>
          )}
          {hasVercelIntegration && (
            <Button
              label="Next"
              onClick={() => {
                updateCompletedSteps(currentStepName, {
                  metadata: {
                    completionSource: 'manual',
                    hostingProvider: 'vercel',
                  },
                });
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: { type: 'btn-click', label: 'next', hostingProvider: 'vercel' },
                });
                router.push(pathCreator.onboardingSteps({ step: nextStepName }));
              }}
            />
          )}
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
            their dashboard.{' '}
            <Link
              size="small"
              href="https://developers.cloudflare.com/workers/configuration/environment-variables/"
              className="inline-block"
              target="_blank"
            >
              Learn more about how to define the variables
            </Link>
          </p>
          <div className="text-basis mb-2 text-sm font-medium">Event key</div>
          <Secret kind="event-key" secret={defaultEventKey} className="mb-4" />
          <div className="text-basis mb-2 text-sm font-medium">Signing key</div>
          <Secret kind="signing-key" secret={env.webhookSigningKey} className="mb-6" />
          <Button
            label="Next"
            onClick={() => {
              updateCompletedSteps(currentStepName, {
                metadata: {
                  completionSource: 'manual',
                  hostingProvider: 'cloudflare',
                },
              });
              tracking?.trackOnboardingAction(currentStepName, {
                metadata: { type: 'btn-click', label: 'next', hostingProvider: 'cloudflare' },
              });
              router.push(pathCreator.onboardingSteps({ step: nextStepName }) + '?nonVercel=true');
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
            You can configure the environment variables on Fly.io by adding the values below.{' '}
            <Link
              size="small"
              href="https://fly.io/docs/rails/the-basics/configuration/"
              className="inline-block"
              target="_blank"
            >
              Learn more about how to set a secret in Fly.io
            </Link>
          </p>
          <div className="text-basis mb-2 text-sm font-medium">Event key</div>
          <Secret
            kind="command"
            secret={`fly secrets set INNGEST_EVENT_KEY=${defaultEventKey}`}
            className="mb-4"
          />
          <div className="text-basis mb-2 text-sm font-medium">Signing key</div>
          <Secret
            kind="command"
            secret={`fly secrets set INNGEST_SIGNING_KEY=${env.webhookSigningKey}`}
            className="mb-6"
          />
          <Button
            label="Next"
            onClick={() => {
              updateCompletedSteps(currentStepName, {
                metadata: {
                  completionSource: 'manual',
                  hostingProvider: 'flyio',
                },
              });
              tracking?.trackOnboardingAction(currentStepName, {
                metadata: { type: 'btn-click', label: 'next', hostingProvider: 'flyio' },
              });
              router.push(pathCreator.onboardingSteps({ step: nextStepName }) + '?nonVercel=true');
            }}
          />
        </TabCards.Content>
      </TabCards>
    </div>
  );
}
