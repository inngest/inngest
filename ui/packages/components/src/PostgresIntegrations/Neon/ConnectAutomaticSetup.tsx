import { useState } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/utils';

import { StatusIndicator, type Step } from './Connect';
import {
  AccessCommand,
  AlterTableReplicationCommandOne,
  AlterTableReplicationCommandTwo,
  ReplicationSlotCommand,
  RoleCommand,
} from './ConnectCommands';

const AutomaticSetup = ({
  onSuccess,
  savedCredentials,
  verifyAutoSetup,
  handleLostCredentials,
}: {
  onSuccess: () => void;
  savedCredentials?: string;
  verifyAutoSetup: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{
    success: boolean;
    error: string;
    steps: {
      [key in Step]: { complete: boolean };
    };
  }>;
  handleLostCredentials: () => void;
}) => {
  const [isVerifying, setIsVerifying] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState<string>();
  const [completionStatus, setCompletionStatus] = useState<{
    logical_replication_enabled: boolean | null;
    publication_created: boolean | null;
    replication_slot_created: boolean | null;
    roles_granted: boolean | null;
    user_created: boolean | null;
  }>({
    logical_replication_enabled: null,
    publication_created: null,
    replication_slot_created: null,
    roles_granted: null,
    user_created: null,
  });

  const handleVerify = async () => {
    setIsVerifying(true);
    setError(undefined);

    if (!savedCredentials) {
      handleLostCredentials();
      return;
    }

    const parsedInput = parseConnectionString(savedCredentials);

    if (!parsedInput) {
      setError('Invalid connection string format. Please check your input.');
      setIsVerifying(false);
      return;
    }

    try {
      const { success, error, steps } = await verifyAutoSetup(parsedInput);
      setCompletionStatus({
        logical_replication_enabled: steps.logical_replication_enabled.complete,
        publication_created: steps.publication_created.complete,
        replication_slot_created: steps.replication_slot_created.complete,
        roles_granted: steps.roles_granted.complete,
        user_created: steps.user_created.complete,
      });
      if (success) {
        setIsVerified(true);
        onSuccess();
      } else {
        setError(error || 'Connection error.');
      }
    } catch (err) {
      setError('An error occurred while connecting. Please try again.');
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <div>
      <p className="text-sm">
        Inngest can setup and connect to your Neon Database automatically. Click the button below
        and we will run a few lines of code for you to make your set up easier.
      </p>
      <div className="my-6">
        <p className="mb-3 font-medium">Inngest will automatically perform the following setup:</p>
        <AccordionList type="multiple" defaultValue={[]}>
          <AccordionList.Item value="user_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a Postgres role for replication</p>
                <StatusIndicator
                  loading={isVerifying}
                  success={completionStatus.user_created === true}
                  error={completionStatus.user_created === false}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Create a dedicated Postgres role for replicating data. The role must have the{' '}
                <code className="text-accent-xIntense text-xs">REPLICATION</code> privilege. The
                default Postgres role created with your Neon project and roles created using the
                Neon CLI, Console, or API are granted membership in the{' '}
                <code className="text-link text-xs">neon_superuser</code> role, which has the
                required <code className="text-accent-xIntense text-xs">REPLICATION</code>{' '}
                privilege.
              </p>
              <RoleCommand />
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="roles_granted">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Grant schema access to your Postgres role</p>
                <StatusIndicator
                  loading={isVerifying}
                  success={completionStatus.roles_granted === true}
                  error={completionStatus.roles_granted === false}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Granting{' '}
                <code className="text-accent-xIntense text-xs">SELECT ON ALL TABLES IN SCHEMA</code>{' '}
                instead of naming the specific tables avoids having to add privileges later if you
                add tables to your publication.
              </p>
              <AccessCommand />
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="replication_slot_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a replication slot</p>
                <StatusIndicator
                  loading={isVerifying}
                  success={completionStatus.replication_slot_created === true}
                  error={completionStatus.replication_slot_created === false}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Inngest uses the <code className="text-accent-xIntense text-xs">pgoutput</code>{' '}
                plugin in Postgres for decoding WAL changes into a logical replication stream. To
                create a replication slot called{' '}
                <code className="text-accent-xIntense text-xs">inngest_slot</code> that uses the{' '}
                <code className="text-accent-xIntense text-xs">pgoutput</code> plugin:
              </p>
              <ReplicationSlotCommand />
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="publication_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a publication</p>
                <StatusIndicator
                  loading={isVerifying}
                  success={completionStatus.publication_created === true}
                  error={completionStatus.publication_created === false}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <ol className="list-decimal pl-10">
                <li className="mb-3">
                  Add the replication identity for each table you want to replicate:
                </li>
                <AlterTableReplicationCommandOne />
                <li className="mb-3 mt-6">Create the Postgres publication:</li>
                <AlterTableReplicationCommandTwo />
              </ol>
              <p className="mt-3">
                The publication name is customizable. Refer to the{' '}
                <NewLink
                  className="inline-block"
                  size="small"
                  href="https://neon.tech/docs/guides/logical-replication-manage"
                >
                  Postgres Docs
                </NewLink>{' '}
                if you need to add or remove tables from your publication.
              </p>
            </AccordionList.Content>
          </AccordionList.Item>
        </AccordionList>
      </div>
      {isVerified ? (
        <NewButton label="See integration" href={`/settings/integrations/neon`} />
      ) : (
        <NewButton
          label="Complete setup automatically"
          onClick={handleVerify}
          loading={isVerifying}
        />
      )}
      {error && <p className="text-error mt-4 text-sm">{error}</p>}
    </div>
  );
};

export default AutomaticSetup;
