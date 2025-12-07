import { useEffect } from "react";
import { useMutation } from "urql";

import ConnectingView from "@/components/DatadogIntegration/ConnectingView";
import { graphql } from "@/gql";

const StartDatadogIntegrationDocument = graphql(`
  mutation StartDatadogIntegration($ddSite: String!, $ddDomain: String!) {
    datadogOAuthRedirectURL(ddSite: $ddSite, ddDomain: $ddDomain)
  }
`);

type StartPageProps = {
  site: string | undefined;
  domain: string | undefined;
};

export default function StartPage({ site, domain }: StartPageProps) {
  const [{ data, error }, startDdInt] = useMutation(
    StartDatadogIntegrationDocument,
  );
  const ddSite = site;
  const ddDomain = domain;
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
