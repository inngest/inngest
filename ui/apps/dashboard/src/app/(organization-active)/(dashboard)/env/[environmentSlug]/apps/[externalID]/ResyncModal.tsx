import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Modal } from '@inngest/components/Modal';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import { RiLoopLeftLine } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import type { CodedError } from '@/codedError';
import { useEnvironment } from '@/components/Environments/environment-context';
import { SyncFailure } from '@/components/SyncFailure/SyncFailure';
import { graphql } from '@/gql';

const ResyncAppDocument = graphql(`
  mutation ResyncApp($appExternalID: String!, $appURL: String, $envID: UUID!) {
    resyncApp(appExternalID: $appExternalID, appURL: $appURL, envID: $envID) {
      app {
        id
      }
      error {
        code
        data
        message
      }
    }
  }
`);

type Props = {
  appExternalID: string;
  isOpen: boolean;
  onClose: () => void;
  url: string;
  platform: string | null;
};

export default function ResyncModal({ appExternalID, isOpen, onClose, url, platform }: Props) {
  const [overrideValue, setOverrideValue] = useState(url);
  const [isURLOverridden, setURLOverridden] = useState(false);
  const [failure, setFailure] = useState<CodedError>();
  const [isSyncing, setIsSyncing] = useState(false);
  const env = useEnvironment();
  const [, resyncApp] = useMutation(ResyncAppDocument);

  if (isURLOverridden) {
    url = overrideValue;
  }

  async function onSync() {
    setIsSyncing(true);

    try {
      const res = await resyncApp(
        {
          appExternalID,
          appURL: url,
          envID: env.id,
        },
        {
          additionalTypenames: [
            // Bust the cache for the Workflow type to prevent the functions
            // list from being stale
            'Workflow',
          ],
        }
      );
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('No API response data');
      }

      if (res.data.resyncApp.error) {
        setFailure(res.data.resyncApp.error);
        return;
      }

      setFailure(undefined);
      toast.success('Synced app');
      onClose();
    } catch (error) {
      setFailure({
        code: 'unknown',
      });
    } finally {
      setIsSyncing(false);
    }
  }

  return (
    <Modal
      className="w-[800px]"
      description="Send a new sync request to your app"
      isOpen={isOpen}
      onClose={onClose}
      title={
        <div className="mb-4 flex flex-row items-center gap-3">
          <RiLoopLeftLine className="h-6 w-6" />
          <h2 className="text-lg font-medium">Resync App</h2>
        </div>
      }
    >
      <div className="border-subtle border-b px-6">
        {platform === 'vercel' && !failure && (
          <Alert className="my-6" severity="info" showIcon={false}>
            Vercel generates a unique URL for each deployment (
            <Alert.Link
              severity="info"
              className="inline"
              href="https://vercel.com/docs/deployments/generated-urls"
              target="_blank"
            >
              see docs
            </Alert.Link>
            ). Please confirm that you are using the correct URL if you choose a deployment&apos;s
            generated URL instead of a static domain for your app.
          </Alert>
        )}
        <p className="my-6">
          This initiates the sync request to your app which pushes the updated function
          configuration to Inngest.
        </p>

        <p className="mb-2">The URL where you serve Inngest functions:</p>

        <div className="mb-6 flex-1">
          <Input
            placeholder="https://example.com/api/inngest"
            name="url"
            value={url}
            onChange={(e) => {
              setOverrideValue(e.target.value);
            }}
            readOnly={!isURLOverridden}
            className={cn(!isURLOverridden && 'bg-disabled')}
          />
        </div>
        <div className="mb-6">
          <SwitchWrapper>
            <Switch
              checked={isURLOverridden}
              disabled={isSyncing}
              onCheckedChange={() => {
                setURLOverridden((prev) => !prev);
              }}
              id="override"
            />
            <SwitchLabel htmlFor="override">Override Input</SwitchLabel>
          </SwitchWrapper>
          {isURLOverridden && !failure && (
            <p className="text-warning pl-[50px] pt-1">
              Please ensure that your app ID (
              <Alert.Link
                severity="warning"
                size="medium"
                className="inline"
                href="https://www.inngest.com/docs/apps#apps-in-sdk"
              >
                docs
              </Alert.Link>
              ) is not changed before resyncing. Changing the app ID will result in the creation of
              a new app in this environment.
            </p>
          )}
        </div>

        {failure && !isSyncing && <SyncFailure error={failure} />}
      </div>

      <div className="flex justify-end gap-2 p-6">
        <Button
          appearance="outlined"
          kind="secondary"
          onClick={onClose}
          disabled={isSyncing}
          label="Cancel"
        />

        <Button onClick={onSync} disabled={isSyncing} kind="primary" label="Resync App" />
      </div>
    </Modal>
  );
}
