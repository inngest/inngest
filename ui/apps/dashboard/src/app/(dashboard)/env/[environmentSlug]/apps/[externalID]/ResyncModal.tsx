import { useState } from 'react';
import ArrowPathIcon from '@heroicons/react/20/solid/ArrowPathIcon';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { Modal } from '@inngest/components/Modal';
import { Switch, SwitchLabel, SwitchWrapper } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import { toast } from 'sonner';

import { Alert } from '@/components/Alert';
import Input from '@/components/Forms/Input';
import { DeployFailure } from '../../deploys/DeployFailure';
import { deployViaUrl, type RegistrationFailure } from '../../deploys/utils';

type Props = {
  isOpen: boolean;
  onClose: () => void;
  url: string;
  platform: string;
};

export default function ResyncModal({ isOpen, onClose, url, platform }: Props) {
  const [overrideValue, setOverrideValue] = useState(url);
  const [isURLOverridden, setURLOverridden] = useState(false);
  const [failure, setFailure] = useState<RegistrationFailure>();
  const [isSyncing, setIsSyncing] = useState(false);

  if (isURLOverridden) {
    url = overrideValue;
  }

  async function onSync() {
    setIsSyncing(true);

    let failure;
    try {
      // TODO: This component is using legacy syncs stuff that needs
      // reorginization and/or refactoring. We should use a GraphQL mutation
      // that gets the last sync URL, rather than relying on the UI to find it.
      failure = await deployViaUrl(url);

      setFailure(failure);
      if (!failure) {
        toast.success('Synced app');
        onClose();
      }
    } catch {
      setFailure({
        errorCode: undefined,
        headers: {},
        statusCode: undefined,
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
          <ArrowPathIcon className="h-6 w-6" />
          <h2 className="text-lg font-medium">Resync App</h2>
        </div>
      }
    >
      <div className="border-b border-slate-200 px-6">
        {platform === 'vercel' && !failure && (
          <Alert className="my-2" severity="info" showIcon={false}>
            Vercel&apos;s Generated URLs (
            <Link showIcon={false} href="(https://vercel.com/docs/deployments/generated-urls">
              see docs
            </Link>
            ) are unique to a particular deployment. Ensure you are using the right URL if you
            choose to use a Generated URL instead of a static domain to locate the deployment.
          </Alert>
        )}
        <p className="my-6">
          This will send a sync request to your app, telling it to sync itself with Inngest.
        </p>

        <p className="my-6">{"We'll"} send a request to the following URL:</p>

        <div className="my-6 flex-1">
          <Input
            placeholder="https://example.com/api/inngest"
            name="url"
            value={url}
            onChange={(e) => {
              setOverrideValue(e.target.value);
            }}
            readonly={!isURLOverridden}
            className={cn(!isURLOverridden && 'bg-slate-200')}
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
            <p className="pl-[50px] pt-1 font-semibold text-yellow-700">
              Ensure your app ID in the new endpoint is the same, otherwise Inngest will consider it
              a new app while syncing.
            </p>
          )}
        </div>

        {failure && !isSyncing && <DeployFailure {...failure} />}
      </div>

      <div className="flex flex-row justify-end gap-4 p-6">
        <Button
          appearance="outlined"
          btnAction={onClose}
          className="px-16"
          disabled={isSyncing}
          label="Cancel"
        />

        <Button
          btnAction={onSync}
          className="px-16"
          disabled={isSyncing}
          kind="primary"
          label="Resync App"
        />
      </div>
    </Modal>
  );
}
