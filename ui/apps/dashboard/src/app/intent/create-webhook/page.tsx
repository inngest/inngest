'use client';

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { ArrowRightIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import slugify from '@sindresorhus/slugify';
import { useMutation } from 'urql';

import { Alert } from '@/components/Alert';
import { graphql } from '@/gql';
import InngestLogo from '@/icons/InngestLogo';
import WebhookIcon from '@/icons/webhookIcon.svg';
import { useEnvironments } from '@/queries';
import { getProductionEnvironment } from '@/utils/environments';
import { createTransform } from '../../(dashboard)/env/[environmentSlug]/manage/[ingestKeys]/[keyID]/TransformEvent';

const CreateWebhook = graphql(`
  mutation CreateWebhook($input: NewIngestKey!) {
    key: createIngestKey(input: $input) {
      id
      url
    }
  }
`);

export default function Page() {
  // TODO - handle failure to fetch environments
  const [{ data: environments }] = useEnvironments();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<String>('');

  const params = useSearchParams();
  const [, createWebhook] = useMutation(CreateWebhook);

  // Params and validation
  const name = params.get('name');
  const redirectURI = params.get('redirect_uri');
  useEffect(() => {
    if (!name) {
      setError('Malformed URL: Missing name parameter');
    }
    if (!redirectURI) {
      setError('Malformed URL: Missing redirect_uri parameter');
    }
  }, [name, redirectURI]);
  const redirectURL: URL | null = useMemo(() => {
    if (!redirectURI) return null;
    try {
      return new URL(redirectURI);
    } catch (e) {
      setError('Malformed URL: redirect_uri is not a valid URL');
    }
    return null;
  }, [redirectURI]);

  const prefix = slugify(name || '');
  const transform = createTransform({
    eventName: `\`${prefix}/\${evt.type}\``,
    dataParam: 'evt.data',
    commentBlock: `// This was created by the ${name} integration.
    // Edit this to customize the event name and payload.`,
  });

  async function approve() {
    const productionEnv = getProductionEnvironment(environments || []);
    if (!productionEnv) {
      throw new Error('Failed to fetch production environment ID');
    }
    setLoading(true);

    createWebhook({
      input: {
        workspaceID: productionEnv.id,
        name: name || '',
        source: 'webhook',
        metadata: {
          transform,
        },
      },
    }).then((result) => {
      setLoading(false);
      if (result.error) {
        setError(result.error.message);
        console.log('error', result.error);
      } else {
        // NOTE - Locally this URL is just a pathname, but in production it's a full URL
        const webhookURL = result.data?.key.url;
        if (!webhookURL || !redirectURL) {
          setError('Failed to create webhook');
          return;
        }
        redirectURL.searchParams.set('url', webhookURL);
        window.location.replace(redirectURL.toString());
      }
    });
  }

  function cancel() {
    if (!redirectURL) {
      return setError('Failed to redirect to redirect_uri');
    }
    redirectURL.searchParams.set('error', 'user_cancelled');
    window.location.replace(redirectURL.toString());
  }

  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto flex h-full max-w-screen-xl flex-col px-6">
        <header className="flex items-center justify-between py-6">
          <InngestLogo />
          <h1 className="hidden">Inngest</h1>
        </header>
        <div className="flex grow items-center">
          <main className="m-auto max-w-2xl pb-24 text-center font-medium">
            <h2 className="my-6 text-xl font-bold">
              {name} is requesting permission to create a new webhook URL
            </h2>
            <div className="my-12 flex flex-row place-content-center items-center justify-items-center gap-6">
              <ArrowRightIcon className="w-16 text-indigo-400" />
              <WebhookIcon className="w-16 text-indigo-400" />
            </div>
            <div className="mx-auto max-w-xl">
              <p className="my-6">
                This will create a new webhook within your <u>Production</u> environment. It can be
                modified or deleted at any time from the Inngest dashboard.
              </p>
              <p className="my-6">
                Upon creation, the webhook will begin sending events with the following prefix:{' '}
                <br />
                <br />
                <pre>
                  {prefix}
                  {'/*'}
                </pre>
              </p>
              <p className="my-6"></p>
            </div>
            <div className="my-12 flex justify-center gap-6">
              <Button
                btnAction={cancel}
                appearance="outlined"
                size="large"
                disabled={loading}
                label="Cancel"
              />
              <Button
                btnAction={approve}
                kind="primary"
                size="large"
                disabled={loading}
                label="Approve"
              />
            </div>
            {error && <Alert severity="error">{error}</Alert>}
            <p className="mt-12 text-sm text-slate-500">
              By approving this request, the created webhook URL will be shared with {name}. <br />
              No other data from your Inngest account will be shared.
            </p>
          </main>
        </div>
      </div>
    </div>
  );
}
