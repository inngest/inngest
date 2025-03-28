'use client';

import { useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import { useMutation } from 'urql';

import ConnectingView from '@/components/DatadogIntegration/ConnectingView';
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

  return <ConnectingView errorMessage={error?.message} />;
}
