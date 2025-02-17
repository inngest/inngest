import { useState } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/utils';
import { IconSpinner } from '@inngest/components/icons/Spinner';
import { RiCheckboxCircleFill, RiCloseCircleFill } from '@remixicon/react';

import {
  AccessCommand,
  AlterTableReplicationCommandOne,
  AlterTableReplicationCommandTwo,
  ReplicationSlotCommand,
  RoleCommand,
} from './ConnectCommands';

const StatusIndicator = ({
  loading,
  success,
  error,
}: {
  loading?: boolean;
  success?: boolean;
  error?: boolean;
}) => {
  if (loading)
    return (
      <div className="text-link flex items-center gap-1 text-sm">
        <IconSpinner className="fill-link h-4 w-4" />
        In progress
      </div>
    );
  if (success) return <RiCheckboxCircleFill className="text-success h-4 w-4" />;
  if (error) return <RiCloseCircleFill className="text-error h-5 w-5" />;
};

export default function Connect({
  onSuccess,
  savedCredentials,
  verifyAutoSetup,
  handleLostCredentials,
  integration,
}: {
  onSuccess: () => void;
  handleLostCredentials: () => void;
  savedCredentials?: string;
  integration: string;
  verifyAutoSetup: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{
    success: boolean;
    error: string;
    steps: {
      logical_replication_enabled: {
        complete: boolean;
      };
      publication_created: {
        complete: boolean;
      };
      replication_slot_created: {
        complete: boolean;
      };
      roles_granted: {
        complete: boolean;
      };
      user_created: {
        complete: boolean;
      };
    };
  }>;
}) {
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

    const parsedInput = parseConnectionString(integration, savedCredentials);

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
    <>
      <p className="text-sm">
        Inngest can setup and connect to your database automatically. Click the button below and we
        will run a few lines of code for you to make your set up easier.
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
                Create a dedicated Inngest role for replicating data. The role must have the{' '}
                <InlineCode>REPLICATION</InlineCode> privilege:
              </p>
              <RoleCommand />
            </AccordionList.Content>
          </AccordionList.Item>
          <AccordionList.Item value="roles_granted">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Grant schema access to your new role:</p>
                <StatusIndicator
                  loading={isVerifying}
                  success={completionStatus.roles_granted === true}
                  error={completionStatus.roles_granted === false}
                />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Grant <InlineCode>SELECT ON ALL TABLES IN SCHEMA</InlineCode> instead of naming the
                specific tables avoids having to add privileges later if you add tables to your
                publication:
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
                Inngest uses the <InlineCode>pgoutput</InlineCode> plugin in Postgres for decoding
                WAL changes into a logical replication stream. To create a replication slot called{' '}
                <InlineCode>inngest_slot</InlineCode> that uses the{' '}
                <InlineCode>pgoutput</InlineCode> plugin, run the following command on your database
                using your replication role:
              </p>
              <ReplicationSlotCommand />
              <p className="mt-3">
                <InlineCode>inngest_slot</InlineCode> is the name assigned to the replication slot.
                You will need to provide this name when you set up your Inngest events.
              </p>
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
              <ol className="list-decimal pb-2 pl-10">
                <li className="mb-3">
                  Create the Postgres publication. Include all tables you want to replicate as part
                  of the publication:
                </li>
                <AlterTableReplicationCommandTwo />
                <li className="mb-3 mt-6">
                  Add the replication identity (the method of distinguishing between rows) for each
                  table you want to replicate:
                </li>
                <AlterTableReplicationCommandOne />
              </ol>
            </AccordionList.Content>
          </AccordionList.Item>
        </AccordionList>
      </div>

      {isVerified ? (
        <Button
          label="Connected. See your integration"
          href={`/settings/integrations/${integration}`}
        />
      ) : (
        <Button label="Complete setup automatically" onClick={handleVerify} loading={isVerifying} />
      )}
      {error && <p className="text-error mt-4 text-sm">{error}</p>}
    </>
  );
}
