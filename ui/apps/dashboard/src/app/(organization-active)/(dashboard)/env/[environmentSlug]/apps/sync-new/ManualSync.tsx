'use client';

import { useMemo, useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Code } from '@inngest/components/Code';
import { Link } from '@inngest/components/Link';
import { useLocalStorage } from 'react-use';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import type { CodedError } from '@/codedError';
import Input from '@/components/Forms/Input';
import { Secret } from '@/components/Secret';
import { SyncFailure } from '@/components/SyncFailure';
import { graphql } from '@/gql';
import { pathCreator } from '@/utils/urls';
import { useEnvironment } from '../../../../../../../components/Environments/old/environment-context';

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
      <div className="border-b border-slate-200 p-8">
        <p>
          To integrate your code hosted on another platform with Inngest, you need to inform Inngest
          about the location of your app and functions.
        </p>
        <br />
        <p>
          For example, imagine that your <Code>serve()</Code> handler (
          <Link
            showIcon={false}
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
          . Verify that you assigned the signing key below to the <Code>INNGEST_SIGNING_KEY</Code>{' '}
          environment variable:
        </p>

        <Secret className="my-6" kind="event-key" secret={env.webhookSigningKey} />

        <div className="border-t border-slate-200">
          <label htmlFor="url" className="my-2 block text-slate-500">
            App URL
          </label>
          <Input
            placeholder="https://example.com/api/inngest"
            name="url"
            value={input}
            onChange={(e) => setInput(e.target.value)}
          />

          {failure && !isSyncing && <SyncFailure error={failure} />}
        </div>
      </div>
      <div className="flex items-center justify-between px-8 py-6">
        <Link href="https://www.inngest.com/docs/apps/cloud">View Docs</Link>
        <div className="flex items-center gap-3">
          <Button
            label="Cancel"
            btnAction={() => {
              router.push(appsURL);
            }}
            appearance="outlined"
          />
          <Button
            label="Sync App"
            btnAction={onSync}
            kind="primary"
            disabled={disabled}
            loading={isSyncing}
          />
        </div>
      </div>
    </>
  );
}
