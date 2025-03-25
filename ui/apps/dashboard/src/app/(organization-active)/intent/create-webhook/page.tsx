'use client';

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Input } from '@inngest/components/Forms/Input';
import { RiArrowRightLine } from '@remixicon/react';
import slugify from '@sindresorhus/slugify';
import { capitalCase } from 'change-case';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import WebhookIcon from '@/icons/webhookIcon.svg';
import { useDefaultEnvironment } from '@/queries';
import { createTransform } from '../../(dashboard)/env/[environmentSlug]/manage/[ingestKeys]/[keyID]/TransformEvent';
import ApprovalDialog from '../ApprovalDialog';

const CreateWebhook = graphql(`
  mutation CreateWebhook($input: NewIngestKey!) {
    key: createIngestKey(input: $input) {
      id
      url
    }
  }
`);

function getNameFromDomain(domain: string | null) {
  if (!domain) return '';
  const removeTLD = domain.replace(/\.com|\.io|\.org|\.net|\.co\..{2}|\.dev|\.app|\.ai|\.xyz$/, '');
  const removeSubdomains = removeTLD.replace(/.*\./, '');
  return removeSubdomains;
}

export default function Page() {
  // TODO - handle failure to fetch environments
  const [{ data: defaultEnv }] = useDefaultEnvironment();
  const [loading, setLoading] = useState(false);
  const [isEditing, setEditing] = useState(false);
  const [customPrefix, setCustomPrefix] = useState<string>('');
  const [error, setError] = useState<string>('');

  const params = useSearchParams();
  const [, createWebhook] = useMutation(CreateWebhook);

  // Params and validation
  const name = params.get('name');
  const domain = params.get('domain');
  const redirectURI = params.get('redirect_uri');
  useEffect(() => {
    if (!name && !domain) {
      setError('Malformed URL: Missing name or domain parameter');
    }
    if (!redirectURI) {
      setError('Malformed URL: Missing redirect_uri parameter');
    }
  }, [name, domain, redirectURI]);
  const redirectURL: URL | null = useMemo(() => {
    if (!redirectURI) return null;
    try {
      return new URL(redirectURI);
    } catch (e) {
      setError('Malformed URL: redirect_uri is not a valid URL');
    }
    return null;
  }, [redirectURI]);

  const displayName = capitalCase(
    name ?? domain !== null ? getNameFromDomain(domain) : 'Webhook integration'
  );

  const slugifyOptions = { preserveCharacters: ['.'] };
  const defaultPrefix = slugify(displayName, slugifyOptions);
  const eventNamePrefix =
    customPrefix !== '' ? slugify(customPrefix, slugifyOptions) : defaultPrefix;

  const transform = createTransform({
    // Svix webhooks do not have a standard schema, so we use fields that
    // are popular with a fallback
    eventName: `\`${eventNamePrefix}/\${evt.type || evt.name || evt.event_type || "webhook.received"}\``,
    // Most webhooks have a data field, but not all, so we fallback to the entire event
    dataParam: 'evt.data || evt',
    commentBlock: `// This was created by the ${displayName} integration.
    // Edit this to customize the event name and payload.`,
  });

  async function approve() {
    if (!defaultEnv) {
      throw new Error('Failed to fetch production environment ID');
    }
    setLoading(true);

    createWebhook({
      input: {
        workspaceID: defaultEnv.id,
        name: displayName,
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
    <ApprovalDialog
      title={`${displayName} is requesting permission to create a new webhook URL`}
      description={
        <>
          <p className="my-6">
            This will create a new webhook within your <u>Production</u> environment. It can be
            modified or deleted at any time from the Inngest dashboard.
          </p>
          <p className="my-6">
            Upon creation, the webhook will begin sending events with the following prefix:{' '}
          </p>
          <div className="flex min-h-[32px] items-center justify-center gap-2">
            {isEditing ? (
              <Input
                type="text"
                placeholder="Add a prefix"
                value={customPrefix}
                onChange={(e) => setCustomPrefix(e.target.value)}
                className="block max-w-[192px]"
              />
            ) : (
              <pre>
                {eventNamePrefix}
                {'/*'}
              </pre>
            )}
            <button
              className="text-muted text-sm"
              onClick={() => {
                setEditing(!isEditing);
                if (customPrefix === '') {
                  setCustomPrefix(defaultPrefix);
                }
              }}
            >
              {isEditing ? 'Save' : 'Edit'}
            </button>
          </div>
        </>
      }
      graphic={
        <>
          <RiArrowRightLine className="text-muted w-16" />
          <WebhookIcon className="text-muted w-16" />
        </>
      }
      isLoading={loading}
      onApprove={approve}
      onCancel={cancel}
      error={error}
      secondaryInfo={
        <>
          By approving this request, the created webhook URL will be shared with {displayName}.{' '}
          <br />
          No other data from your Inngest account will be shared.
        </>
      }
    />
  );
}
