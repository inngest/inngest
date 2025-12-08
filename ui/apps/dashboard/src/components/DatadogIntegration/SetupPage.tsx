"use client";

import { useState } from "react";
import { Alert } from "@inngest/components/Alert";
import { Button } from "@inngest/components/Button";
import { AlertModal } from "@inngest/components/Modal";
import { StatusDot } from "@inngest/components/Status/StatusDot";
import { Time } from "@inngest/components/Time";
import { IconSpinner } from "@inngest/components/icons/Spinner";
import { RiOrganizationChart } from "@remixicon/react";
import { toast } from "sonner";
import { useMutation, useQuery } from "urql";

import { graphql } from "@/gql";
import type { DatadogConnectionStatus } from "@/gql/graphql";

export const ddIntegrationHref =
  "https://app.datadoghq.com/integrations/inngest";

export const GetDatadogSetupDataDocument = graphql(`
  query GetDatadogSetupData {
    account {
      datadogConnections {
        id
        orgID
        orgName
        envID
        envName
        healthy
        lastErrorMessage
        lastSentAt
      }
      datadogOrganizations {
        id
        datadogDomain
        datadogOrgName
      }
    }
  }
`);

const DisableDatadogConnectionDocument = graphql(`
  mutation DisableDatadogConnection($connectionID: UUID!) {
    disableDatadogConnection(connectionID: $connectionID)
  }
`);

const RemoveDatadogOrganizationDocument = graphql(`
  mutation RemoveDatadogOrganization($organizationID: UUID!) {
    removeDatadogOrganization(organizationID: $organizationID)
  }
`);

type ConnectionToDisable = {
  envName: string;
  connectionID: string;
  orgName: string;
};

type OrganizationToRemove = {
  orgName: string;
  organizationID: string;
};

function dotStatusForConnection(conn: DatadogConnectionStatus) {
  if (conn.healthy) {
    return "ACTIVE";
  } else {
    return "FAILED";
  }
}

function alertTextBeforeRemovingOrg(
  org: OrganizationToRemove | null,
  allConnections: DatadogConnectionStatus[],
): string {
  if (!org) {
    return "";
  }
  const affectedConns = allConnections.filter(
    (conn) => conn.orgID == org.organizationID,
  );
  let text =
    "Are you certain you want to disconnect Inngest from the “" +
    org.orgName +
    "” Datadog organization?";
  if (affectedConns.length == 1) {
    text +=
      " This will disable sending metrics from your “" +
      affectedConns[0]?.envName +
      "” environment.";
  } else if (affectedConns.length > 1) {
    text +=
      " This will disable sending metrics from " +
      affectedConns.length +
      " environments.";
  }
  return text;
}

export default function SetupPage({}) {
  const [{ data: ddSetupData, error: ddSetupFetchError }, refetchDdSetupData] =
    useQuery({
      query: GetDatadogSetupDataDocument,
    });
  const [, disableConnection] = useMutation(DisableDatadogConnectionDocument);
  const [, removeOrganization] = useMutation(RemoveDatadogOrganizationDocument);

  const [selectedConnToDisable, setSelectedConnToDisable] =
    useState<ConnectionToDisable | null>(null);
  const [selectedOrgToRemove, setSelectedOrgToRemove] =
    useState<OrganizationToRemove | null>(null);

  const commitRemoveSelectedOrganization = async () => {
    if (!selectedOrgToRemove) {
      return;
    }

    const result = await removeOrganization(
      { organizationID: selectedOrgToRemove.organizationID },
      {
        additionalTypenames: ["DatadogOrganization", "DatadogConnectionStatus"],
      },
    );
    setSelectedOrgToRemove(null);
    refetchDdSetupData();

    if (result.error) {
      toast.error(`Failed: ${result.error}`);
      console.error(result.error);
      return;
    }
  };

  const commitDisableSelectedConnection = async () => {
    if (!selectedConnToDisable) {
      return;
    }

    const result = await disableConnection(
      { connectionID: selectedConnToDisable.connectionID },
      { additionalTypenames: ["DatadogConnectionStatus"] },
    );
    setSelectedConnToDisable(null);
    refetchDdSetupData();

    if (result.error) {
      toast.error(`Failed: ${result.error}`);
      console.error(result.error);
      return;
    }
  };

  if (ddSetupFetchError) {
    console.error(ddSetupFetchError);
  }

  const connectEnvHref = "/settings/integrations/datadog/connect-env";

  return (
    <>
      <AlertModal
        isOpen={selectedConnToDisable !== null}
        onClose={() => setSelectedConnToDisable(null)}
        title={
          "Disable sending metrics from “" +
          selectedConnToDisable?.envName +
          "” to “" +
          selectedConnToDisable?.orgName +
          "”"
        }
        description={
          "Are you sure you want to disable sending metrics to the “" +
          selectedConnToDisable?.orgName +
          "” Datadog organization for the “" +
          selectedConnToDisable?.envName +
          "” environment?"
        }
        confirmButtonLabel="Remove"
        onSubmit={commitDisableSelectedConnection}
      />

      <AlertModal
        isOpen={selectedOrgToRemove !== null}
        onClose={() => setSelectedOrgToRemove(null)}
        title={"Remove “" + selectedOrgToRemove?.orgName + "”"}
        description={alertTextBeforeRemovingOrg(
          selectedOrgToRemove,
          ddSetupData?.account.datadogConnections || [],
        )}
        confirmButtonLabel="Remove"
        onSubmit={commitRemoveSelectedOrganization}
      />

      {ddSetupData && ddSetupData.account.datadogOrganizations.length > 0 && (
        <div className="mb-12">
          <div className="mb-2 flex flex-row justify-start">
            <div className="text-basis flex-1 text-xl font-medium">
              Environments
            </div>
            {ddSetupData.account.datadogConnections.length > 0 && (
              <Button
                appearance="outlined"
                kind="primary"
                label="Connect Environment"
                href={connectEnvHref}
                className="mr-2"
              />
            )}
          </div>

          {ddSetupData.account.datadogConnections.length === 0 && (
            <div className="border-subtle flex flex-col items-center gap-4 rounded border p-8 text-center">
              No environments are sending metrics to Datadog right now.
              <Button
                appearance="solid"
                kind="primary"
                label="Connect Environment"
                href={connectEnvHref}
                className="text-sm"
              />
            </div>
          )}

          {ddSetupData.account.datadogConnections.length > 0 &&
            ddSetupData.account.datadogConnections.map((ddConn, i) => (
              <div
                className={`text-basis flex flex-row justify-start gap-3 border-t px-2 pb-2 pt-3 ${
                  i === ddSetupData.account.datadogConnections.length - 1
                    ? "border-b"
                    : ""
                }`}
                key={i}
              >
                <StatusDot status={dotStatusForConnection(ddConn)} />
                <div className="-mt-1 flex flex-1 flex-col">
                  <div>
                    <span className="font-medium">{ddConn.envName}</span>
                  </div>
                  {ddConn.lastErrorMessage ? (
                    <div className="font-normal">
                      <span className="text-error font-bold">Error:</span>{" "}
                      <code>{ddConn.lastErrorMessage}</code>
                    </div>
                  ) : ddConn.lastSentAt ? (
                    <div className="text-muted">
                      <span className="italic">Metrics last sent:</span>{" "}
                      <Time value={ddConn.lastSentAt} />
                    </div>
                  ) : ddConn.healthy ? (
                    <div className="text-muted">
                      <span className="italic">Setting up…</span>
                    </div>
                  ) : (
                    <div className="font-normal">
                      <span className="text-error font-bold">Error:</span>{" "}
                      <code>Please contact Inngest Support.</code>
                    </div>
                  )}
                </div>
                {ddSetupData.account.datadogOrganizations.length > 1 && (
                  <div className="mr-1 mt-1">
                    <span className="text-muted italic">connected to</span>{" "}
                    <span className="font-medium">{ddConn.orgName}</span>
                  </div>
                )}
                <div>
                  <Button
                    appearance="outlined"
                    kind="danger"
                    label="Disable"
                    onClick={() => {
                      setSelectedConnToDisable({
                        envName: ddConn.envName,
                        orgName: ddConn.orgName,
                        connectionID: ddConn.id,
                      });
                    }}
                  />
                </div>
              </div>
            ))}
        </div>
      )}

      <div className="mb-12">
        <div className="mb-2 flex flex-row justify-start">
          <div className="text-basis flex-1 text-xl font-medium">
            {!ddSetupData ||
            ddSetupData.account.datadogOrganizations.length === 0
              ? "Datadog Connections"
              : "Connected Datadog Organizations"}
          </div>
          {ddSetupData &&
            ddSetupData.account.datadogOrganizations.length > 0 && (
              <Button
                appearance="outlined"
                kind="primary"
                label="Add Organization"
                href={ddIntegrationHref}
                className="mr-2"
              />
            )}
        </div>

        {ddSetupFetchError && (
          <Alert severity="error" className="mx-auto mb-3 mt-3">
            <p className="text-balance">
              An error occurred when communicating with Inngest; please refresh
              this page.
            </p>
          </Alert>
        )}

        {!ddSetupData && !ddSetupFetchError && (
          <IconSpinner className="fill-link h-8 w-8 text-center" />
        )}

        {ddSetupData &&
          ddSetupData.account.datadogOrganizations.length === 0 && (
            <div className="border-subtle flex flex-col items-center gap-4 rounded border p-8 text-center">
              Inngest isn’t connected to Datadog yet.
              <Button
                appearance="solid"
                kind="primary"
                label="Connect to Datadog"
                href={ddIntegrationHref}
                className="text-sm"
              />
            </div>
          )}

        {ddSetupData &&
          ddSetupData.account.datadogOrganizations.length > 0 &&
          ddSetupData.account.datadogOrganizations.map((ddOrg, i) => (
            <div
              className={`text-basis flex flex-row justify-start gap-2 border-t px-2 pb-2 pt-3 ${
                i === ddSetupData.account.datadogOrganizations.length - 1
                  ? "border-b"
                  : ""
              }`}
              key={i}
            >
              <RiOrganizationChart className="text-muted h-4 w-4" />
              <div className="-mt-1 flex flex-1 flex-col">
                <div>
                  <span className="font-medium">{ddOrg.datadogOrgName}</span>
                </div>
                <div className="text-muted">
                  <code>{ddOrg.datadogDomain}</code>
                </div>
              </div>
              <div>
                <Button
                  appearance="outlined"
                  kind="danger"
                  label="Remove"
                  onClick={() => {
                    setSelectedOrgToRemove({
                      orgName: ddOrg.datadogOrgName || ddOrg.id,
                      organizationID: ddOrg.id,
                    });
                  }}
                />
              </div>
            </div>
          ))}
      </div>
    </>
  );
}
