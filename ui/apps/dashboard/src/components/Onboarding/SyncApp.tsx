import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { RiCheckboxCircleFill, RiCloseCircleFill, RiInputCursorMove } from '@remixicon/react';

import { type CodedError } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import { SyncFailure } from '../SyncFailure';
import CommonVercelErrors from './CommonVercelErrors';
import { getVercelSyncs, syncAppManually, type VercelSyncsResponse } from './actions';
import { type VercelApp } from './data';
import useOnboardingStep from './useOnboardingStep';
import { useOnboardingTracking } from './useOnboardingTracking';
import { getNextStepName } from './utils';

export default function SyncApp() {
  const currentStepName = OnboardingSteps.SyncApp;
  const nextStepName = getNextStepName(currentStepName);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingVercelApps, setIsLoadingVercelApps] = useState(false);
  const [vercelSyncs, setVercelSyncs] = useState<VercelSyncsResponse>();
  const [error, setError] = useState<CodedError | null>();
  const [app, setApp] = useState<string | null>();
  const { updateCompletedSteps } = useOnboardingStep();
  const router = useRouter();
  const tracking = useOnboardingTracking();

  const searchParams = useSearchParams();
  const fromNonVercel = searchParams.get('nonVercel') === 'true';

  const loadVercelSyncs = async () => {
    try {
      setIsLoadingVercelApps(true);
      const syncs = await getVercelSyncs();
      setVercelSyncs(syncs);
      return syncs;
    } catch (err) {
      console.error('Failed to load syncs: ', err);
    } finally {
      setIsLoadingVercelApps(false);
    }
  };

  useEffect(() => {
    let intervalId: number | undefined;

    const checkAndPoll = async () => {
      const syncs = await loadVercelSyncs();

      const hasPendingSync = syncs?.apps.some(
        (app: VercelApp) => app.latestSync?.status === 'pending'
      );

      if (hasPendingSync) {
        intervalId = window.setInterval(async () => {
          const newSyncs = await loadVercelSyncs();
          const newHasPendingSync = newSyncs?.apps.some(
            (app: VercelApp) => app.latestSync?.status === 'pending'
          );

          // Clear interval if the pending syncs are resolved
          if (!newHasPendingSync && intervalId !== undefined) {
            window.clearInterval(intervalId);
          }
        }, 2500);
      }
    };

    checkAndPoll();

    return () => {
      if (intervalId !== undefined) {
        window.clearInterval(intervalId);
      }
    };
  }, []);

  console.log('vercel', vercelSyncs);

  const hasSuccessfulSync = vercelSyncs?.apps.some(
    (app) => app.latestSync?.status === 'success' || app.latestSync?.status === 'duplicate'
  );

  const handleSyncAppManually = async () => {
    setIsLoading(true);
    setError(null);
    setApp('');
    try {
      const { success, error, appName } = await syncAppManually(inputValue);
      if (success) {
        setApp(appName);
        updateCompletedSteps(currentStepName, {
          metadata: {
            completionSource: 'manual',
            syncMethod: 'manual',
          },
        });
      } else {
        setError(error);
      }
    } catch (err) {
      setError({ message: 'An unexpected error occurred', code: '', data: '' });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        Since your code is hosted on another platform, you need to register where your functions are
        hosted with Inngest.{' '}
        <Link
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/apps/cloud?ref=app-onboarding-sync-app"
          target="_blank"
        >
          Learn more about syncs
        </Link>
      </p>

      <h4 className="mb-4 text-sm font-medium">Choose syncing method:</h4>
      <TabCards defaultValue={fromNonVercel ? 'manually' : 'vercel'}>
        <TabCards.ButtonList>
          <TabCards.Button className="w-36" value="manually">
            <div className="flex items-center gap-1.5">
              <RiInputCursorMove className="h-5 w-5" /> Sync manually
            </div>
          </TabCards.Button>
          <TabCards.Button className="w-36" value="vercel">
            <div className="flex items-center gap-1.5">
              <IconVercel className="h-4 w-4" /> Sync with Vercel
            </div>
          </TabCards.Button>
        </TabCards.ButtonList>
        <TabCards.Content value="manually">
          <div className="mb-4 flex items-center gap-2">
            <div className="bg-canvasBase border-muted flex h-9 w-9 items-center justify-center rounded border">
              <RiInputCursorMove className="text-muted h-4 w-4" />
            </div>
            <p className="text-basis">Sync your app manually</p>
          </div>
          <p className="mb-4 text-sm">
            Enter the URL of your application&apos;s serve endpoint to register your functions with
            Inngest.
          </p>
          <Alert severity="info">
            <p className="text-sm">
              If you set up the serve handler at /api/inngest, and your domain is https://myapp.com,
              you&apos;ll need to inform Inngest that your app is hosted at
              https://myapp.com/api/inngest.
            </p>
            <Alert.Link
              severity="info"
              className=""
              size="small"
              href="https://www.inngest.com/docs/reference/serve?ref=app-onboarding-sync-app"
              target="_blank"
            >
              Learn more about serve()
            </Alert.Link>
          </Alert>
          <Input
            className={`${error && 'outline-error'} my-3 w-full`}
            placeholder="https://myapp.com/api/inngest"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
          />
          {app && (
            <div className="border-subtle mb-4 flex items-center justify-between rounded-md border p-3">
              <div className="flex items-center gap-1.5">
                <div className="bg-contrast border-muted flex h-9 w-9 items-center justify-center rounded border">
                  <AppsIcon className="text-onContrast h-4 w-4" />
                </div>
                <div>{app}</div>
              </div>
              <div className="text-primary-moderate flex items-center gap-1.5 text-sm">
                <RiCheckboxCircleFill className="text-primary-moderate h-4 w-4" />
                App synced successfully
              </div>
            </div>
          )}
          {error && <SyncFailure className="mb-3 mt-0 text-sm" error={error} />}
          {!app && (
            <div className="flex items-center gap-2">
              <Button
                loading={isLoading}
                label="Sync app here"
                onClick={() => {
                  tracking?.trackOnboardingAction(currentStepName, {
                    metadata: { type: 'btn-click', label: 'sync', syncMethod: 'manual' },
                  });
                  handleSyncAppManually();
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
          )}
          {app && (
            <Button
              label="Next"
              onClick={() => {
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: { type: 'btn-click', label: 'next', syncMethod: 'manual' },
                });
                router.push(pathCreator.onboardingSteps({ step: nextStepName }));
              }}
            />
          )}
        </TabCards.Content>
        <TabCards.Content value="vercel">
          <div className="mb-4 flex items-center justify-between gap-1">
            <div className="flex items-center gap-2">
              <div className="bg-canvasMuted flex h-9 w-9 items-center justify-center rounded">
                <IconVercel className="text-basis h-4 w-4" />
              </div>
              <p className="text-basis">Vercel</p>
            </div>
            <Button
              kind="secondary"
              appearance="outlined"
              label="Manage Vercel integration"
              href={pathCreator.vercel()}
              size="small"
              onClick={() =>
                tracking?.trackOnboardingAction(currentStepName, {
                  metadata: {
                    type: 'btn-click',
                    label: 'view-integration',
                    syncMethod: 'vercel',
                  },
                })
              }
            />
          </div>
          <p className="mb-4 text-sm">
            Inngest <span className="font-medium">automatically</span> syncs your app upon
            deployment, ensuring a seamless connection.
          </p>
          {isLoadingVercelApps && !vercelSyncs && (
            <div className="text-link mb-4 flex items-center gap-1 text-sm">
              <IconSpinner className="fill-link h-4 w-4" />
              Loading apps
            </div>
          )}
          {vercelSyncs && (
            <>
              {vercelSyncs.apps.length ? (
                <div>
                  {vercelSyncs.apps.map((app) => (
                    <div
                      key={app.id}
                      className="border-subtle mb-4 flex items-center justify-between rounded border p-3"
                    >
                      {app.name && (
                        <div className="flex items-center gap-2">
                          <div className="bg-contrast border-muted flex h-9 w-9 items-center justify-center rounded border">
                            <AppsIcon className="text-onContrast h-4 w-4" />
                          </div>
                          <p className="text-basis">{app.name}</p>
                        </div>
                      )}
                      <StatusIndicator status={app.latestSync?.status} />
                    </div>
                  ))}
                </div>
              ) : vercelSyncs.unattachedSyncs.length ? (
                <>
                  <SyncFailure
                    className="mb-4"
                    error={{
                      message: vercelSyncs.unattachedSyncs[0]?.error || 'Unknown error',
                      code: 'unknown',
                    }}
                  />
                  <CommonVercelErrors />
                </>
              ) : (
                <div className="mb-4">
                  <div className="border-subtle mb-4 flex items-center justify-between rounded-md border p-3 text-sm">
                    No syncs found
                  </div>
                  <CommonVercelErrors />
                </div>
              )}
            </>
          )}
          <Button
            disabled={!hasSuccessfulSync}
            label="Next"
            onClick={() => {
              updateCompletedSteps(currentStepName, {
                metadata: {
                  completionSource: 'manual',
                  syncMethod: 'vercel',
                },
              });
              tracking?.trackOnboardingAction(currentStepName, {
                metadata: { type: 'btn-click', label: 'next', syncMethod: 'vercel' },
              });
              router.push(pathCreator.onboardingSteps({ step: nextStepName }));
            }}
          />
        </TabCards.Content>
      </TabCards>
    </div>
  );
}

const StatusIndicator = ({ status }: { status?: string }) => {
  if (status === 'pending')
    return (
      <div className="text-link flex items-center gap-1 text-sm">
        <IconSpinner className="fill-link h-4 w-4" />
        Syncing app
      </div>
    );
  if (status === 'success')
    return (
      <div className="text-success flex items-center gap-1 text-sm">
        <RiCheckboxCircleFill className="text-success h-4 w-4" />
        App synced successfully
      </div>
    );
  if (status === 'error')
    return (
      <div className="text-error flex items-center gap-1 text-sm">
        <RiCloseCircleFill className="text-error h-5 w-5" />
        App failed to sync
      </div>
    );
  if (status === 'duplicate')
    return (
      <div className="text-success flex items-center gap-1 text-sm">
        <RiCheckboxCircleFill className="text-success h-4 w-4" />
        App synced successfully
      </div>
    );
  return <></>;
};
