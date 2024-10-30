import { useState } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/utils';

import { StatusIndicator, type Step } from './Connect';
import {
  AccessCommandManual,
  AlterTableReplicationCommandOneFullManual,
  AlterTableReplicationCommandOneManual,
  AlterTableReplicationCommandTwoFullManual,
  AlterTableReplicationCommandTwoManual,
  ReplicationSlotCommandManual,
  RoleCommandManual,
} from './ConnectCommands';

interface StepState {
  isVerifying: boolean;
  error?: string;
  isComplete: boolean;
}

const ManualSetup = ({
  onSuccess,
  savedCredentials,
  verifyManualSetup,
  handleLostCredentials,
}: {
  onSuccess: () => void;
  savedCredentials?: string;
  verifyManualSetup: (variables: {
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
  const [stepStates, setStepStates] = useState<Record<Step, StepState>>({
    logical_replication_enabled: { isVerifying: false, isComplete: false },
    user_created: { isVerifying: false, isComplete: false },
    roles_granted: { isVerifying: false, isComplete: false },
    replication_slot_created: { isVerifying: false, isComplete: false },
    publication_created: { isVerifying: false, isComplete: false },
  });

  const handleVerifyStep = async (step: Step) => {
    if (!savedCredentials) {
      handleLostCredentials();
      return;
    }

    setStepStates((prev) => ({
      ...prev,
      [step]: { ...prev[step], isVerifying: true, error: undefined },
    }));

    const parsedInput = parseConnectionString(savedCredentials);
    if (!parsedInput) {
      setStepStates((prev) => ({
        ...prev,
        [step]: {
          ...prev[step],
          isVerifying: false,
          error: 'Invalid connection string format.',
        },
      }));
      return;
    }

    try {
      const { success, error, steps } = await verifyManualSetup(parsedInput);
      setStepStates((prev) => ({
        ...prev,
        [step]: {
          isVerifying: false,
          isComplete: steps[step]?.complete || false,
          error: !success ? error : undefined,
        },
      }));
      if (success) {
        const allStepsComplete = Object.values(steps).every((state) => state.complete);
        if (allStepsComplete) {
          onSuccess();
        }
      }
    } catch (err) {
      setStepStates((prev) => ({
        ...prev,
        [step]: {
          ...prev[step],
          isVerifying: false,
          error: 'An error occurred while verifying. Please try again.',
        },
      }));
    }
  };

  const allStepsComplete = Object.values(stepStates).every((state) => state.isComplete);

  return (
    <div>
      <p className="text-sm">
        Follow the steps below to manually connect your Neon database to Inngest:
      </p>
      <div className="my-6">
        <AccordionList type="multiple" defaultValue={['user_created']}>
          <AccordionList.Item value="user_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a Postgres role for replication</p>
                <StatusIndicator
                  loading={stepStates.user_created.isVerifying}
                  success={stepStates.user_created.isComplete}
                  error={!!stepStates.user_created.error}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                It's recommended that you create a dedicated Postgres role for replicating data. The
                role must have the <code className="text-accent-xIntense text-xs">REPLICATION</code>{' '}
                privilege. The default Postgres role created with your Neon project and roles
                created using the Neon CLI, Console, or API are granted membership in the{' '}
                <code className="text-link text-xs">neon_superuser</code> role, which has the
                required <code className="text-accent-xIntense text-xs">REPLICATION</code>{' '}
                privilege.
              </p>
              <RoleCommandManual />
              <div className="items-top mt-3 flex justify-between">
                <div>
                  {stepStates.user_created.error && (
                    <p className="text-error text-sm">{stepStates.user_created.error}</p>
                  )}
                </div>
                <NewButton
                  appearance="solid"
                  label="Validate step"
                  onClick={() => handleVerifyStep('user_created')}
                  loading={stepStates.user_created.isVerifying}
                />
              </div>
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="roles_granted">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Grant schema access to your Postgres role</p>
                <StatusIndicator
                  loading={stepStates.roles_granted.isVerifying}
                  success={stepStates.roles_granted.isComplete}
                  error={!!stepStates.roles_granted.error}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                If your replication role does not own the schemas and tables you are replicating
                from, make sure to grant access. For example, the following commands grant access to
                all tables in the <code className="text-accent-xIntense text-xs">public</code>{' '}
                schema to Postgres role{' '}
                <code className="text-accent-xIntense text-xs">replication_user</code>:
              </p>
              <AccessCommandManual />
              <p className="my-3">
                Granting{' '}
                <code className="text-accent-xIntense text-xs">SELECT ON ALL TABLES IN SCHEMA</code>{' '}
                instead of naming the specific tables avoids having to add privileges later if you
                add tables to your publication.
              </p>
              <div className="items-top mt-3 flex justify-between">
                <div>
                  {stepStates.roles_granted.error && (
                    <p className="text-error text-sm">{stepStates.roles_granted.error}</p>
                  )}
                </div>
                <NewButton
                  appearance="solid"
                  label="Validate step"
                  onClick={() => handleVerifyStep('roles_granted')}
                  loading={stepStates.roles_granted.isVerifying}
                />
              </div>
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="replication_slot_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a replication slot</p>
                <StatusIndicator
                  loading={stepStates.replication_slot_created.isVerifying}
                  success={stepStates.replication_slot_created.isComplete}
                  error={!!stepStates.replication_slot_created.error}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Inngest requires a dedicated replication slot. Only one source should be configured
                to use this replication slot.
              </p>
              <p className="mb-3">
                Inngest uses the <code className="text-accent-xIntense text-xs">pgoutput</code>{' '}
                plugin in Postgres for decoding WAL changes into a logical replication stream. To
                create a replication slot called{' '}
                <code className="text-accent-xIntense text-xs">inngest_slot</code> that uses the{' '}
                <code className="text-accent-xIntense text-xs">pgoutput</code> plugin, run the
                following command on your database using your replication role:
              </p>
              <ReplicationSlotCommandManual />
              <p className="my-3">
                <code className="text-accent-xIntense text-xs">inngest_slot</code> is the name
                assigned to the replication slot. You will need to provide this name when you set up
                your Inngest events.
              </p>
              <div className="items-top mt-3 flex justify-between">
                <div>
                  {stepStates.replication_slot_created.error && (
                    <p className="text-error text-sm">
                      {stepStates.replication_slot_created.error}
                    </p>
                  )}
                </div>
                <NewButton
                  appearance="solid"
                  label="Validate step"
                  onClick={() => handleVerifyStep('replication_slot_created')}
                  loading={stepStates.replication_slot_created.isVerifying}
                />
              </div>
            </AccordionList.Content>
          </AccordionList.Item>

          <AccordionList.Item value="publication_created">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a publication</p>
                <StatusIndicator
                  loading={stepStates.publication_created.isVerifying}
                  success={stepStates.publication_created.isComplete}
                  error={!!stepStates.publication_created.error}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Perform the following steps for each table you want to replicate data from:
              </p>
              <ol className="list-decimal pl-10">
                <li className="mb-3">
                  Add the replication identity (the method of distinguishing between rows) for each
                  table you want to replicate:
                </li>
                <AlterTableReplicationCommandOneManual />
                <p className="my-3">
                  In rare cases, if your tables use data types that support{' '}
                  <code className="text-accent-xIntense text-xs">TOAST</code> or have very large
                  field values, consider using{' '}
                  <code className="text-accent-xIntense text-xs">REPLICA IDENTITY FULL</code>{' '}
                  instead:
                </p>
                <AlterTableReplicationCommandOneFullManual />
                <li className="mb-3 mt-6">
                  Create the Postgres publication. Include all tables you want to replicate as part
                  of the publication:
                </li>
                <AlterTableReplicationCommandTwoManual />
                <p className="my-3">Alternatively, you can create a publication for all tables:</p>
                <AlterTableReplicationCommandTwoFullManual />
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
              </ol>
              <div className="items-top mt-3 flex justify-between">
                <div>
                  {stepStates.publication_created.error && (
                    <p className="text-error text-sm">{stepStates.publication_created.error}</p>
                  )}
                </div>
                <NewButton
                  appearance="solid"
                  label="Validate step"
                  onClick={() => handleVerifyStep('publication_created')}
                  loading={stepStates.publication_created.isVerifying}
                />
              </div>
            </AccordionList.Content>
          </AccordionList.Item>
        </AccordionList>
      </div>
      <NewButton
        label="Complete setup"
        href={`/settings/integrations/neon`}
        disabled={!allStepsComplete}
      />
    </div>
  );
};

export default ManualSetup;
