import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert } from '@inngest/components/Alert/Alert';
import { NewButton } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { NewLink } from '@inngest/components/Link';
import TabCards from '@inngest/components/TabCards/TabCards';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { RiCheckboxCircleFill, RiInputCursorMove } from '@remixicon/react';

import { type CodedError } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { OnboardingSteps } from '../Onboarding/types';
import { syncAppManually } from './actions';
import useOnboardingStep from './useOnboardingStep';

export default function SyncApp() {
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<CodedError | null>();
  const [app, setApp] = useState<string | null>();
  const { updateLastCompletedStep } = useOnboardingStep();
  const router = useRouter();

  const handleSyncAppManually = async () => {
    setIsLoading(true);
    setError(null);
    setApp('');
    try {
      const { success, error, appName } = await syncAppManually(inputValue);
      if (success) {
        setApp(appName);
        updateLastCompletedStep(OnboardingSteps.SyncApp);
      } else {
        setError(error);
      }
    } catch (err) {
      // setError({message: 'An unexpected error occurred'});
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="text-subtle">
      <p className="mb-6 text-sm">
        Since your code is hosted on another platform, you need to register where your functions are
        hosted with Inngest.{' '}
        <NewLink
          className="inline-block"
          size="small"
          href="https://www.inngest.com/docs/apps/cloud?ref=app-onboarding-sync-app"
        >
          Learn more about syncs
        </NewLink>
      </p>

      <h4 className="mb-4 text-sm font-medium">Choose syncing method:</h4>
      <TabCards defaultValue="manually">
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
            >
              Learn more about serve()
            </Alert.Link>
          </Alert>
          <Input
            className={`${error && 'outline-error'} my-3 `}
            placeholder="https://myapp.com/api/inngest"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
          />
          {app && (
            <div className="border-subtle flex items-center justify-between rounded-md border p-3">
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
          {/* TODO: Add advanced error structure */}
          {error && (
            <Alert className="mb-3 text-sm" severity="error">
              {error.message}
            </Alert>
          )}
          {!app && (
            <NewButton loading={isLoading} label="Sync app here" onClick={handleSyncAppManually} />
          )}
          {app && (
            <NewButton
              label="Next"
              onClick={() => {
                router.push(pathCreator.onboardingSteps({ step: OnboardingSteps.InvokeFn }));
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
            <NewButton
              kind="secondary"
              appearance="outlined"
              label="View Vercel dashboard"
              href={pathCreator.vercel()}
              size="small"
            />
          </div>
          <p className="mb-4 text-sm">
            Inngest <span className="font-medium">automatically</span> syncs your app upon
            deployment, ensuring a seamless connection.
          </p>
          {/* TODO: wire vercel integration flow */}
          <div className="text-link mb-4 flex items-center gap-1 text-sm">
            <IconSpinner className="fill-link h-4 w-4" />
            Syncing app
          </div>
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
