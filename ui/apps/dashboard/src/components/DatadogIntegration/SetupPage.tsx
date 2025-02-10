'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card/Card';
import { Input } from '@inngest/components/Forms/Input';
import { StatusDot } from '@inngest/components/Status/StatusDot';
import { Time } from '@inngest/components/Time';
import { IconDatadog } from '@inngest/components/icons/platforms/Datadog';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import IntegrationNotEnabledMessage from '@/components/IntegrationNotEnabledMessage';
import EnvSelectMenu from '@/components/PrometheusIntegration/EnvSelectMenu';
import { graphql } from '@/gql';
import type { DatadogIntegration } from '@/gql/graphql';
import { useEnvironments } from '@/queries';
import type { Environment } from '@/utils/environments';

type Props = {
  metricsExportEnabled: boolean;
};

const GetDatadogIntegrationDocument = graphql(`
  query GetDatadogIntegration($workspaceID: ID!) {
    workspace(id: $workspaceID) {
      datadogIntegration {
        id
        datadogSite
        appKey
        appKeyUpdatedAt
        apiKey
        apiKeyUpdatedAt
      }
    }
  }
`);

const ListDatadogIntegrationsDocument = graphql(`
  query ListDatadogIntegrations {
    account {
      datadogIntegrations {
        id
        accountID
        envID
        datadogSite
        appKey
        appKeyUpdatedAt
        apiKey
        apiKeyUpdatedAt
        lastSentAt
        createdAt
        updatedAt
        statusOk
      }
    }
  }
`);

const SetupDatadogIntegrationDocument = graphql(`
  mutation SetupDatadogIntegration(
    $workspaceID: UUID!
    $apiKey: String!
    $appKey: String!
    $ddSite: String!
  ) {
    setupDatadogIntegration(
      envID: $workspaceID
      apiKey: $apiKey
      appKey: $appKey
      datadogSite: $ddSite
    ) {
      integration {
        id
      }
      error
    }
  }
`);

function dotStatusForIntegration(integration: DatadogIntegration) {
  if (integration.statusOk) {
    return 'ACTIVE';
  } else {
    return 'FAILED';
  }
}

function findEnvName(envs: Environment[], id: string) {
  const env = envs.find((env) => env.id === id);
  return env ? env.name : id;
}

export default function SetupPage({ metricsExportEnabled }: Props) {
  const [{ data: envs = [], error: envsErr }] = useEnvironments();
  const [selectedEnv, setSelectedEnv] = useState<Environment | null>(null);
  const selectedEnvName = selectedEnv ? selectedEnv.name : '';
  const [{ data: ddIntData, fetching: ddIntFetching }, refetchDdInt] = useQuery({
    query: GetDatadogIntegrationDocument,
    variables: {
      workspaceID: selectedEnv?.id || '',
    },
    pause: !selectedEnv,
  });
  const [{ data: allDatadogInts }] = useQuery({
    query: ListDatadogIntegrationsDocument,
  });
  const [, setupDdInt] = useMutation(SetupDatadogIntegrationDocument);
  const [isFormDisabled, setFormDisabled] = useState(false);
  const [formError, setFormError] = useState('');

  if (envsErr) {
    console.error('error fetching envs', envsErr);
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormDisabled(true);

    const form = new FormData(event.currentTarget);
    const appKey = form.get('appKey') as string | null;
    if (!appKey) {
      return;
    }
    const apiKey = form.get('apiKey') as string | null;
    if (!apiKey) {
      return;
    }
    const ddSite = form.get('datadogSite') as string | null;

    const result = await setupDdInt({
      workspaceID: selectedEnv?.id || '',
      appKey,
      apiKey,
      ddSite: ddSite || '',
    });
    setFormDisabled(false);

    if (result.error) {
      toast.error(`Failed: ${result.error}`);
      setFormError(result.error.message);
      console.error(result.error);
      return;
    }

    if (result.data?.setupDatadogIntegration?.error) {
      setFormError(result.data.setupDatadogIntegration.error);
      return;
    }

    event.target.reset();
    refetchDdInt();
    toast.success(`Datadog integration configured for ${selectedEnvName}`);
    setFormError('');
    setSelectedEnv(selectedEnv);
  }

  const onEnvSelect = (env: Environment) => {
    setSelectedEnv(env);
    setFormError('');
  };

  const isNewIntegration = !ddIntData?.workspace.datadogIntegration;

  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
          <IconDatadog className="text-onContrast" size={20} />
        </div>
        Datadog
      </div>

      <div className="text-muted mb-6 w-full text-base font-normal">
        Connect to send key Inngest metrics directly to your Datadog account.
        {/* TODO: Link to Datadog docs, once we've written them */}
        {/*<Link target="_blank" size="medium" href="https://www.inngest.com/docs/deploy/vercel">*/}
        {/*  Read documentation*/}
        {/*</Link>*/}
      </div>

      {!metricsExportEnabled && <IntegrationNotEnabledMessage integrationName="Datadog" />}

      {metricsExportEnabled && (
        <div className="text-sm font-normal">
          {allDatadogInts && allDatadogInts.account.datadogIntegrations.length > 0 && (
            <div className="mb-10">
              <div className="text-basis mb-4 justify-start text-xl font-medium">
                Integration Status
              </div>
              {allDatadogInts.account.datadogIntegrations.map((ddInt, i) => (
                <div
                  className={`text-basis flex flex-row justify-start gap-3 border-t px-2 pb-2 pt-3 ${
                    i === allDatadogInts.account.datadogIntegrations.length - 1 ? 'border-b' : ''
                  }`}
                  key={i}
                >
                  <StatusDot status={dotStatusForIntegration(ddInt)}></StatusDot>
                  <div className="-mt-1 flex flex-col">
                    <div>
                      <span className="font-medium">{findEnvName(envs, ddInt.envID)}</span>
                    </div>
                    {ddInt.lastSentAt && (
                      <div className="text-muted">
                        <span className="italic">Metrics last sent:</span>{' '}
                        <Time value={ddInt.lastSentAt} />
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          <div className="text-basis mb-4 justify-start text-xl font-medium">Configuration</div>
          <Card
            accentColor={formError !== '' ? 'bg-errorContrast' : 'bg-surfaceMuted'}
            accentPosition="left"
            className="w-full"
          >
            <Card.Header>
              <div className="text-basis mb-1 text-sm">
                Select an environment to manage its Datadog integration:
              </div>
              <EnvSelectMenu onSelect={onEnvSelect} className="mb-2" />
            </Card.Header>
            <Card.Content>
              <form onSubmit={handleSubmit} className="flex flex-col items-start">
                <label htmlFor="appKey" className="text-muted mt-1">
                  Datadog App Key
                </label>
                <div className="flex flex-row gap-4">
                  <Input
                    className="mb-2 min-w-[300px]"
                    type="text"
                    name="appKey"
                    placeholder=""
                    defaultValue={ddIntData?.workspace.datadogIntegration?.appKey || ''}
                    required
                    disabled={isFormDisabled || ddIntFetching}
                  />
                  {!isNewIntegration && ddIntData.workspace.datadogIntegration && (
                    <div className="text-muted mt-1 italic">
                      Updated:{' '}
                      <Time value={ddIntData.workspace.datadogIntegration.appKeyUpdatedAt} />
                    </div>
                  )}
                </div>
                <label htmlFor="apiKey" className="text-muted mt-1">
                  Datadog API Key
                </label>
                <div className="flex flex-row gap-4">
                  <Input
                    className="mb-2 min-w-[300px]"
                    type="text"
                    name="apiKey"
                    placeholder=""
                    defaultValue={ddIntData?.workspace.datadogIntegration?.apiKey || ''}
                    required
                    disabled={isFormDisabled || ddIntFetching}
                  />
                  {!isNewIntegration && ddIntData.workspace.datadogIntegration && (
                    <div className="text-muted mt-1 italic">
                      Updated:{' '}
                      <Time value={ddIntData.workspace.datadogIntegration.apiKeyUpdatedAt} />
                    </div>
                  )}
                </div>
                <label htmlFor="datadogSite" className="text-muted mt-1">
                  Datadog Site
                </label>
                <div className="flex flex-row gap-4">
                  <Input
                    className="mb-2 min-w-[300px]"
                    type="text"
                    name="datadogSite"
                    placeholder="datadoghq.com"
                    defaultValue={ddIntData?.workspace.datadogIntegration?.datadogSite || ''}
                    disabled={isFormDisabled || ddIntFetching}
                  />
                  <Button
                    kind="primary"
                    type="submit"
                    disabled={isFormDisabled || ddIntFetching}
                    label={isNewIntegration ? 'Create integration' : 'Update integration'}
                  />
                </div>
              </form>
              {formError !== '' && (
                <Alert severity="error" className="mx-auto max-w-xs">
                  <p className="text-balance">{formError}</p>
                </Alert>
              )}
            </Card.Content>
          </Card>
        </div>
      )}
    </div>
  );
}
