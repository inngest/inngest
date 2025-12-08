"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
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

export default function FinishPage({}) {
  const [{ data, error }, finishDdInt] = useMutation(
    FinishDatadogIntegrationDocument,
  );
  const searchParams = useSearchParams();
  const ddSite = searchParams.get("site");
  const ddDomain = searchParams.get("domain");
  const authCode = searchParams.get("code");
  const orgID = searchParams.get("dd_oid");
  const orgName = searchParams.get("dd_org_name");
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
