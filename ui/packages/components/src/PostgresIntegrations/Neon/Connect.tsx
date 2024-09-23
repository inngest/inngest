import { useState } from 'react';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/types';
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
}: {
  onSuccess: () => void;
  savedCredentials?: string;
  verifyAutoSetup: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<boolean>;
}) {
  const [isVerifying, setIsVerifying] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [error, setError] = useState<string>();

  const handleVerify = async () => {
    setIsVerifying(true);
    setError(undefined);

    if (!savedCredentials) {
      setError('Lost credentials. Go back to the first step.');
      setIsVerifying(false);
      return;
    }

    const parsedInput = parseConnectionString(savedCredentials);

    if (!parsedInput) {
      setError('Invalid connection string format. Please check your input.');
      setIsVerifying(false);
      return;
    }

    try {
      const success = await verifyAutoSetup(parsedInput);
      if (success) {
        setIsVerified(true);
        onSuccess();
      } else {
        setError('Connection error.');
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
        Inngest can setup and connect to your Neon Database automatically. Click the button below
        and we will run a few lines of code for you to make your set up easier.
      </p>

      <div className="my-6">
        <p className="mb-3 font-medium">Inngest will automatically perform the following setup:</p>
        <AccordionList type="multiple" defaultValue={[]}>
          <AccordionList.Item value="1">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a Postgres role for replication</p>
                <StatusIndicator loading={isVerifying} success={isVerified} error={!!error} />
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
          <AccordionList.Item value="2">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Grant schema access to your Postgres role</p>
                <StatusIndicator loading={isVerifying} success={isVerified} error={!!error} />
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
          <AccordionList.Item value="3">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a replication slot</p>
                <StatusIndicator loading={isVerifying} success={isVerified} error={!!error} />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <p className="mb-3">
                Inngest uses the <code className="text-accent-xIntense text-xs">pgoutput</code>{' '}
                plugin in Postgres for decoding WAL changes into a logical replication stream. To
                create a replication slot called{' '}
                <code className="text-accent-xIntense text-xs">inngest_slot</code> that uses the{' '}
                <code className="text-accent-xIntense text-xs">pgoutput</code> plugin, run the
                following command on your database using your replication role:
              </p>
              <ReplicationSlotCommand />
              <p className="mt-3">
                <code className="text-accent-xIntense text-xs">inngest_slot</code> is the name
                assigned to the replication slot. You will need to provide this name when you set up
                your Inngest events.
              </p>
            </AccordionList.Content>
          </AccordionList.Item>
          <AccordionList.Item value="4">
            <AccordionList.Trigger>
              <div className="flex w-full items-center justify-between">
                <p>Create a publication</p>
                <StatusIndicator loading={isVerifying} success={isVerified} error={!!error} />
              </div>
            </AccordionList.Trigger>
            <AccordionList.Content>
              <ol className="list-decimal pl-10">
                <li className="mb-3">
                  Add the replication identity (the method of distinguishing between rows) for each
                  table you want to replicate:
                </li>
                <AlterTableReplicationCommandOne />
                <li className="mb-3 mt-6">
                  Create the Postgres publication. Include all tables you want to replicate as part
                  of the publication:
                </li>
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
    </>
  );
}
