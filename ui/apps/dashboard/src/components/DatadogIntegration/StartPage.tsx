"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { useMutation } from "urql";

import ConnectingView from "@/components/DatadogIntegration/ConnectingView";
import { graphql } from "@/gql";

const StartDatadogIntegrationDocument = graphql(`
  mutation StartDatadogIntegration($ddSite: String!, $ddDomain: String!) {
    datadogOAuthRedirectURL(ddSite: $ddSite, ddDomain: $ddDomain)
  }
`);

export default function StartPage({}) {
  const [{ data, error }, startDdInt] = useMutation(
    StartDatadogIntegrationDocument,
  );
  const searchParams = useSearchParams();
  const ddSite = searchParams.get("site");
  const ddDomain = searchParams.get("domain");
  const oauthStateReady = ddSite && ddDomain;

  useEffect(() => {
    if (!oauthStateReady) {
      return;
    }

    startDdInt({
      ddSite: ddSite,
      ddDomain: ddDomain,
    });
  }, [startDdInt, ddSite, ddDomain, oauthStateReady]);

  useEffect(() => {
    if (data) {
      window.location.href = data.datadogOAuthRedirectURL;
    }
  }, [data]);

  if (!oauthStateReady) {
    return (
      <ConnectingView errorMessage="Expected authentication flow parameters are missing. Please try connecting to Datadog again from the beginning." />
    );
  }

  return <ConnectingView errorMessage={error?.message} />;
}
