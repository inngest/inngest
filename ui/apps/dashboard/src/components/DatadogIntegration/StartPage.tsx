'use client';

import { useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import { Alert } from '@inngest/components/Alert';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { useMutation } from 'urql';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import IntegrationNotEnabledMessage from '@/components/Integration/IntegrationNotEnabledMessage';
import { graphql } from '@/gql';

const StartDatadogIntegrationDocument = graphql(`
  mutation StartDatadogIntegration($ddSite: String!, $ddDomain: String!) {
    datadogOAuthRedirectURL(ddSite: $ddSite, ddDomain: $ddDomain)
  }
`);

export default function StartPage({}) {
  const [{ data, error }, startDdInt] = useMutation(StartDatadogIntegrationDocument);
  const { value: ddIntFlagEnabled } = useBooleanFlag('datadog-integration');
  const searchParams = useSearchParams();
  const ddSite = searchParams.get('site');
  const ddDomain = searchParams.get('domain');

  useEffect(() => {
    if (!ddSite || !ddDomain || !ddIntFlagEnabled) {
      return;
    }

    startDdInt({
      ddSite: ddSite,
      ddDomain: ddDomain,
    });
  }, [startDdInt, ddSite, ddDomain, ddIntFlagEnabled]);

  if (!ddIntFlagEnabled) {
    return <IntegrationNotEnabledMessage integrationName="Datadog" />;
  }

  if (data) {
    window.location.href = data.datadogOAuthRedirectURL;
  }

  return (
    <div className="mx-auto mt-16 flex w-[800px]  flex-col">
      {error ? (
        <>
          <Alert severity="error">
            Please{' '}
            <a href="/support" className="underline">
              contact Inngest support
            </a>{' '}
            if this error persists.
            <br />
            <br />
            <code>{error.message}</code>
          </Alert>
        </>
      ) : (
        <>
          <div className="text-link flex items-center gap-1.5 text-sm">
            <IconSpinner className="fill-link h-16 w-16" />
          </div>
        </>
      )}
    </div>
  );
}
