'use client';

import { useMemo, useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link';
import { useLocalStorage } from 'react-use';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import type { CodedError } from '@/codedError';
import { useEnvironment } from '@/components/Environments/environment-context';
import { Secret } from '@/components/Secret';
import { SyncFailure } from '@/components/SyncFailure';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';

const SyncNewAppDocument = graphql(`
  mutation SyncNewApp($appURL: String!, $envID: UUID!) {
    syncNewApp(appURL: $appURL, envID: $envID) {
      app {
        externalID
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
  appsURL: Route;
};

export default function ManualSync({ appsURL }: Props) {
  const [input = '', setInput] = useLocalStorage('deploymentUrl', '');
  const [failure, setFailure] = useState<CodedError>();
  const [isSyncing, setIsSyncing] = useState(false);
  const router = useRouter();
  const env = useEnvironment();
  const [, syncNewApp] = useMutation(SyncNewAppDocument);

  async function onSync() {
    setIsSyncing(true);

    try {
      // TODO: This component is using legacy syncs stuff that needs
      // reorginization and/or refactoring. We should use a GraphQL mutation
      // that gets the last sync URL, rather than relying on the UI to find it.
      // failure = await deployViaUrl(url);
      const res = await syncNewApp({
        appURL: input,
        envID: env.id,
      });
      if (res.error) {
        throw res.error;
      }
      if (!res.data) {
        throw new Error('No API response data');
      }

      if (res.data.syncNewApp.error) {
        setFailure(res.data.syncNewApp.error);
        return;
      }

      setFailure(undefined);
      toast.success('Synced app');

      const { externalID } = res.data.syncNewApp.app ?? {};
      let navURL;
      if (externalID) {
        navURL = pathCreator.app({
          envSlug: env.slug,
          externalAppID: externalID,
        });
      } else {
        // Should be unreachable
        navURL = pathCreator.apps({ envSlug: env.slug });
      }

      router.push(navURL);
    } catch (error) {
      setFailure({
        code: 'unknown',
      });
    } finally {
      setIsSyncing(false);
    }
  }

  /**
   * Disable the button if the URL isn't valid
   */
  const disabled = useMemo(() => {
    try {
      new URL(input);
      return false;
    } catch {
      return true;
    }
  }, [input]);

  return (
    <>
      <p>
        To integrate your code hosted on another platform with Inngest, you need to inform Inngest
        about the location of your app and functions.
      </p>
      <br />
      <p>
        For example, imagine that your <InlineCode>serve()</InlineCode> handler (
        <Link
          size="small"
          className="inline-flex"
          href="https://www.inngest.com/docs/reference/serve#how-the-serve-api-handler-works"
        >
          see docs
        </Link>
        ) is located at /api/inngest, and your domain is myapp.com. In this scenario, you&apos;ll
        need to inform Inngest that your apps and functions are hosted at
        https://myapp.com/api/inngest.
      </p>
      <br />
      <p>
        After you&apos;ve set up the serve API and deployed your code,{' '}
        <span className="font-semibold">
          enter the URL of your project&apos;s serve endpoint to sync your app with Inngest
        </span>
        . Verify that you assigned the signing key below to the{' '}
        <InlineCode>INNGEST_SIGNING_KEY</InlineCode> environment variable:
      </p>

      <Secret className="my-6" kind="event-key" secret={env.webhookSigningKey} />

      <div className="border-subtle border-t">
        <label htmlFor="url" className="text-muted my-2 block">
          App URL
        </label>
        <Input
          placeholder="https://example.com/api/inngest"
          name="url"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="w-full"
        />

        {failure && !isSyncing && <SyncFailure error={failure} />}
      </div>
      <div className="flex items-center justify-between pt-6">
        <Link href="https://www.inngest.com/docs/apps/cloud" target="_blank" size="small">
          View Docs
        </Link>
        <div className="flex items-center gap-3">
          <Button
            label="Cancel"
            kind="secondary"
            onClick={() => {
              router.push(appsURL);
            }}
            appearance="outlined"
          />
          <Button
            label="Sync app"
            onClick={onSync}
            kind="primary"
            disabled={disabled}
            loading={isSyncing}
          />
        </div>
      </div>
    </>
  );
}
