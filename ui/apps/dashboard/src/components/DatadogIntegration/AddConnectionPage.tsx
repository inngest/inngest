import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Alert } from '@inngest/components/Alert/NewAlert';
import { Button } from '@inngest/components/Button/NewButton';
import { Card } from '@inngest/components/Card';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import EnvSelectMenu from '@/components/Environments/EnvSelectMenu';
import { graphql } from '@/gql';
import { useEnvironments } from '@/queries';
import type { Environment } from '@/utils/environments';
import { GetDatadogSetupDataDocument, ddIntegrationHref } from './SetupPage';

const EnableDatadogConnectionDocument = graphql(`
  mutation EnableDatadogConnection($organizationID: UUID!, $envID: UUID!) {
    enableDatadogConnection(organizationID: $organizationID, envID: $envID) {
      id
    }
  }
`);

export default function AddConnectionPage({}) {
  const navigate = useNavigate();

  const [{ data: envs = [], error: envsErr }] = useEnvironments();
  const [{ data: ddSetupData, error: ddSetupErr }] = useQuery({
    query: GetDatadogSetupDataDocument,
  });

  const [selectedEnv, setSelectedEnv] = useState<Environment | null>(null);
  const [isFormDisabled, setFormDisabled] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [, enableDatadogConnection] = useMutation(
    EnableDatadogConnectionDocument,
  );

  const onEnvSelect = (env: Environment) => {
    setSelectedEnv(env);
    setFormError(null);
  };

  if (envsErr) {
    toast.error(`Failed to load: ${envsErr.message}`);
    console.error(envsErr);
    return;
  } else if (ddSetupErr) {
    toast.error(`Failed to load: ${ddSetupErr.message}`);
    console.error(ddSetupErr);
    return;
  }

  if (!ddSetupData || envs.length === 0) {
    return <IconSpinner className="fill-link h-8 w-8 text-center" />;
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const form = new FormData(event.currentTarget);
    const orgID = form.get('selectedOrg') as string | null;
    if (!orgID || !selectedEnv) {
      return;
    }

    setFormDisabled(true);
    const result = await enableDatadogConnection(
      {
        organizationID: orgID,
        envID: selectedEnv.id,
      },
      { additionalTypenames: ['DatadogConnectionStatus'] },
    );

    if (result.error) {
      setFormDisabled(false);
      toast.error(`Failed connecting environment to Datadog`);
      setFormError(result.error.message);
      console.error(result.error);
      return;
    }

    toast.success(`Datadog integration configured for ${selectedEnv.name}`);
    navigate({ to: '/settings/integrations/datadog' });
  }

  const extantConnectionsForEnv = ddSetupData.account.datadogConnections.filter(
    (connection) => {
      return connection.envID === selectedEnv?.id;
    },
  );
  const availableDatadogOrgsForEnv =
    ddSetupData.account.datadogOrganizations.filter((org) => {
      return !extantConnectionsForEnv.some((connection) => {
        return connection.orgID === org.id;
      });
    });

  let cardAccentColor = 'bg-surfaceMuted';
  if (formError) {
    cardAccentColor = 'bg-errorContrast';
  }

  return (
    <Card
      accentColor={cardAccentColor}
      accentPosition="left"
      className="h-full overflow-visible"
      contentClassName="overflow-visible"
    >
      <Card.Header>
        <div className="text-basis mb-1 text-sm">
          Choose an environment to connect to Datadog:
        </div>
        <EnvSelectMenu onSelect={onEnvSelect} className="mb-2" />
      </Card.Header>
      <Card.Content>
        {formError && (
          <Alert severity="error" className="mx-auto mb-3 mt-3">
            <p className="text-balance">{formError}</p>
          </Alert>
        )}

        {availableDatadogOrgsForEnv.length === 0 && (
          <Alert severity="info" className="mx-auto mb-3 mt-3">
            <p className="text-balance">
              <span className="font-semibold">{selectedEnv?.name}</span> is
              already connected to all available Datadog organizations.
            </p>
            <p>
              To connect a new Datadog organization, please{' '}
              <a
                href={ddIntegrationHref}
                className="underline"
                target="_blank"
                rel="noopener noreferrer"
              >
                navigate to the Inngest integration from your Datadog
                organization
              </a>{' '}
              and start the connection process from there.
            </p>
          </Alert>
        )}

        {availableDatadogOrgsForEnv.length > 0 && (
          <form
            onSubmit={handleSubmit}
            className="flex flex-col items-start gap-4"
          >
            <div>Choose the Datadog organization to send metrics to:</div>
            {availableDatadogOrgsForEnv.map((org, i) => (
              <div className="flex flex-row gap-2" key={org.id}>
                <input
                  type="radio"
                  name="selectedOrg"
                  value={org.id}
                  id={org.id}
                  disabled={isFormDisabled}
                  defaultChecked={i === 0}
                />
                <label htmlFor={org.id} className="-mt-0.5">
                  {org.datadogOrgName || org.id}
                </label>
              </div>
            ))}

            <Button
              kind="primary"
              type="submit"
              disabled={isFormDisabled}
              label="Connect"
              className="mt-4"
            />
          </form>
        )}
      </Card.Content>
    </Card>
  );
}
