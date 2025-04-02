'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal';
import { StatusDot } from '@inngest/components/Status/StatusDot';
import { Time } from '@inngest/components/Time';
import { IconDatadog } from '@inngest/components/icons/platforms/Datadog';
import { RiOrganizationChart } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import IntegrationNotEnabledMessage from '@/components/Integration/IntegrationNotEnabledMessage';
import MetricsExportEntitlementBanner from '@/components/Integration/MetricsExportEntitlementsBanner';
import { graphql } from '@/gql';
import type { DatadogConnectionStatus } from '@/gql/graphql';
import { useBooleanFlag } from '../FeatureFlags/hooks';

type Props = {
  metricsExportEnabled: boolean;
  metricsGranularitySeconds: number;
  metricsFreshnessSeconds: number;
};

const GetDatadogSetupDataDocument = graphql(`
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
    return 'ACTIVE';
  } else {
    return 'FAILED';
  }
}

function alertTextBeforeRemovingOrg(
  org: OrganizationToRemove | null,
  allConnections: DatadogConnectionStatus[]
): string {
  if (!org) {
    return '';
  }
  const affectedConns = allConnections.filter((conn) => conn.orgID == org.organizationID);
  let text =
    'Are you certain you want to disconnect Inngest from the “' +
    org.orgName +
    '” Datadog organization?';
  if (affectedConns.length == 1) {
    text +=
      ' This will disable metrics export from the “' + affectedConns[0]?.envName + '” environment.';
  } else if (affectedConns.length > 1) {
    text += ' This will disable metrics export from ' + affectedConns.length + ' environments.';
  }
  return text;
}

export default function SetupPage({
  metricsExportEnabled,
  metricsGranularitySeconds,
  metricsFreshnessSeconds,
}: Props) {
  const [{ data: ddSetupData }, refetchDdSetupData] = useQuery({
    query: GetDatadogSetupDataDocument,
  });
  const [, disableConnection] = useMutation(DisableDatadogConnectionDocument);
  const [, removeOrganization] = useMutation(RemoveDatadogOrganizationDocument);

  const [selectedConnToDisable, setSelectedConnToDisable] = useState<ConnectionToDisable | null>(
    null
  );
  const [selectedOrgToRemove, setSelectedOrgToRemove] = useState<OrganizationToRemove | null>(null);

  if (!ddIntFlagEnabled) {
    return <IntegrationNotEnabledMessage integrationName="Datadog" />;
  }

  const commitRemoveSelectedOrganization = async () => {
    if (!selectedOrgToRemove) {
      return;
    }

    const result = await removeOrganization(
      { organizationID: selectedOrgToRemove.organizationID },
      { additionalTypenames: ['DatadogOrganization', 'DatadogConnectionStatus'] }
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
      { additionalTypenames: ['DatadogConnectionStatus'] }
    );
    setSelectedConnToDisable(null);
    refetchDdSetupData();

    if (result.error) {
      toast.error(`Failed: ${result.error}`);
      console.error(result.error);
      return;
    }
  };

  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconDatadog className="text-onContrast" size={20} />
        </div>
        Datadog
      </div>

      <div className="text-muted mb-6 w-full text-base font-normal">
        Send key Inngest metrics directly to your Datadog account.
        {/* TODO: Link to Datadog docs, once we've written them */}
        {/*<Link target="_blank" size="medium" href="https://www.inngest.com/docs/deploy/vercel">*/}
        {/*  Read documentation*/}
        {/*</Link>*/}
      </div>

      {!metricsExportEnabled && <IntegrationNotEnabledMessage integrationName="Datadog" />}

      <AlertModal
        isOpen={selectedConnToDisable !== null}
        onClose={() => setSelectedConnToDisable(null)}
        title={
          'Disable metrics export from “' +
          selectedConnToDisable?.envName +
          '” to “' +
          selectedConnToDisable?.orgName +
          '”'
        }
        description={
          'Are you sure you want to disable metrics export to the “' +
          selectedConnToDisable?.orgName +
          '” Datadog organization for the “' +
          selectedConnToDisable?.envName +
          '” environment?'
        }
        confirmButtonLabel="Remove"
        onSubmit={commitDisableSelectedConnection}
      />

      <AlertModal
        isOpen={selectedOrgToRemove !== null}
        onClose={() => setSelectedOrgToRemove(null)}
        title={'Remove “' + selectedOrgToRemove?.orgName + '”'}
        description={alertTextBeforeRemovingOrg(
          selectedOrgToRemove,
          ddSetupData?.account.datadogConnections || []
        )}
        confirmButtonLabel="Remove"
        onSubmit={commitRemoveSelectedOrganization}
      />

      {metricsExportEnabled && (
        <div className="text-sm font-normal">
          <MetricsExportEntitlementBanner
            granularitySeconds={metricsGranularitySeconds}
            freshnessSeconds={metricsFreshnessSeconds}
            className={'mb-12'}
          />

          {ddSetupData && ddSetupData.account.datadogConnections.length > 0 && (
            <div className="mb-12">
              <div className="mb-2 flex flex-row justify-start">
                <div className="text-basis flex-1 text-xl font-medium">Environments</div>
                {ddSetupData.account.datadogConnections.length > 0 && (
                  <Button
                    appearance="outlined"
                    kind="primary"
                    label="Connect Environment"
                    href="/settings/integrations/datadog/connect-env"
                    className="mr-2"
                  />
                )}
              </div>
              {/* TODO(cdzombak): Handle "no connections" case */}

              {ddSetupData.account.datadogConnections.length > 0 &&
                ddSetupData.account.datadogConnections.map((ddConn, i) => (
                  <div
                    className={`text-basis flex flex-row justify-start gap-3 border-t px-2 pb-2 pt-3 ${
                      i === ddSetupData.account.datadogConnections.length - 1 ? 'border-b' : ''
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
                          <span className="text-error font-bold">Error:</span>{' '}
                          <code>{ddConn.lastErrorMessage}</code>
                        </div>
                      ) : ddConn.lastSentAt ? (
                        <div className="text-muted">
                          <span className="italic">Metrics last sent:</span>{' '}
                          <Time value={ddConn.lastSentAt} />
                        </div>
                      ) : ddConn.healthy ? (
                        <div className="text-muted">
                          <span className="italic">Setting up…</span>
                        </div>
                      ) : (
                        <div className="font-normal">
                          <span className="text-error font-bold">Error:</span>{' '}
                          <code>Please contact Inngest Support.</code>
                        </div>
                      )}
                    </div>
                    {ddSetupData.account.datadogOrganizations.length > 1 && (
                      <div className="mr-1 mt-1">
                        <span className="text-muted italic">connected to</span>{' '}
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
                Connected Datadog Organizations
              </div>
              {ddSetupData && ddSetupData.account.datadogOrganizations.length > 0 && (
                <Button
                  appearance="outlined"
                  kind="primary"
                  label="Add Organization"
                  href="https://app.datadoghq.com/marketplace"
                  className="mr-2"
                />
              )}
              {/* TODO(cdzombak): update link once Marketplace listing is live */}
            </div>

            {/* TODO(cdzombak): Handle "no organizations" case */}

            {ddSetupData &&
              ddSetupData.account.datadogOrganizations.length > 0 &&
              ddSetupData.account.datadogOrganizations.map((ddOrg, i) => (
                <div
                  className={`text-basis flex flex-row justify-start gap-2 border-t px-2 pb-2 pt-3 ${
                    i === ddSetupData.account.datadogOrganizations.length - 1 ? 'border-b' : ''
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
        </div>
      )}
    </div>
  );
}
