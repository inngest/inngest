import { useEffect } from "react";
import { useMutation } from "urql";

import ConnectingView from "@/components/DatadogIntegration/ConnectingView";
import { graphql } from "@/gql";

const FinishDatadogIntegrationDocument = graphql(`
  mutation FinishDatadogIntegrationDocument(
    $orgName: String!
    $orgID: String!
    $authCode: String!
    $ddSite: String!
    $ddDomain: String!
  ) {
    datadogOAuthCompleted(
      orgName: $orgName
      orgID: $orgID
      authCode: $authCode
      ddSite: $ddSite
      ddDomain: $ddDomain
    ) {
      id
    }
  }
`);

type FinishPageProps = {
  site: string | undefined;
  domain: string | undefined;
  code: string | undefined;
  dd_oid: string | undefined;
  dd_org_name: string | undefined;
};

export default function FinishPage({
  site,
  domain,
  code,
  dd_oid,
  dd_org_name,
}: FinishPageProps) {
  const [{ data, error }, finishDdInt] = useMutation(
    FinishDatadogIntegrationDocument,
  );
  const ddSite = site;
  const ddDomain = domain;
  const authCode = code;
  const orgID = dd_oid;
  const orgName = dd_org_name;
  const oauthStateReady = ddSite && ddDomain && authCode && orgID && orgName;

  useEffect(() => {
    if (!oauthStateReady) {
      return;
    }

    finishDdInt({
      orgName: orgName,
      orgID: orgID,
      authCode: authCode,
      ddSite: ddSite,
      ddDomain: ddDomain,
    });
  }, [
    finishDdInt,
    orgID,
    orgName,
    authCode,
    ddSite,
    ddDomain,
    oauthStateReady,
  ]);

  useEffect(() => {
    if (data) {
      window.location.href = "/settings/integrations/datadog";
    }
  }, [data]);

  if (!oauthStateReady) {
    return (
      <ConnectingView errorMessage="Expected authentication flow parameters are missing. Please try connecting to Datadog again from the beginning." />
    );
  }

  return <ConnectingView errorMessage={error?.message} />;
}
