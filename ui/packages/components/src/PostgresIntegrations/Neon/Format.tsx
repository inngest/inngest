import { useState } from 'react';
import { NewButton } from '@inngest/components/Button';
import { NewLink } from '@inngest/components/Link';
import {
  IntegrationSteps,
  parseConnectionString,
} from '@inngest/components/PostgresIntegrations/types';

export default function NeonFormat({
  onSuccess,
  savedCredentials,
  verifyLogicalReplication,
}: {
  onSuccess: () => void;
  savedCredentials?: string;
  verifyLogicalReplication: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{ success: boolean; error: string }>;
}) {
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState<string>();
  const [isVerified, setIsVerified] = useState(false);

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
      const { success, error } = await verifyLogicalReplication(parsedInput);
      if (success) {
        setIsVerified(true);
        onSuccess();
      } else {
        setError(
          error ||
            'Could not verify credentials. Please check if everything is entered correctly and try again.'
        );
      }
    } catch (err) {
      setError('An error occurred while verifying. Please try again.');
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <>
      <p className="text-sm">
        Enabling logical replication modifies the Postgres{' '}
        <code className="text-accent-xIntense text-xs">wal_level</code> configuration parameter,
        changing it from <code className="text-accent-xIntense text-xs">replica</code> to{' '}
        <code className="text-accent-xIntense text-xs">logical</code> for all databases in your Neon
        project. Once the <code className="text-accent-xIntense text-xs">wal_level</code> setting is
        changed to <code className="text-accent-xIntense text-xs">logical</code>, it cannot be
        reverted. Enabling logical replication also restarts all computes in your Neon project,
        meaning active connections will be dropped and have to reconnect.
      </p>
      <NewLink
        size="small"
        href="https://neon.tech/docs/guides/logical-replication-concepts#write-ahead-log-wal"
      >
        Learn more about WAL level
      </NewLink>

      <div className="my-6">
        <p className="mb-3">To enable logical replication in Neon:</p>
        <ol className="list-decimal pl-10 text-sm leading-8">
          <li>Select your project in the Neon Console.</li>
          <li>
            On the Neon <span className="text-medium">Dashboard</span>, select{' '}
            <span className="text-medium">Settings</span>.
          </li>
          <li>
            Select <span className="text-medium">Logical Replication</span>.
          </li>
          <li>
            Click <span className="text-medium">Enable</span> to enable logical replication.
          </li>
        </ol>
      </div>

      {isVerified ? (
        <NewButton
          label="Next"
          href={`/settings/integrations/neon/${IntegrationSteps.ConnectDb}`}
        />
      ) : (
        <NewButton
          label="Verify logical replication is enabled"
          onClick={handleVerify}
          loading={isVerifying}
        />
      )}
      {error && <p className="text-error mt-4 text-sm">{error}</p>}
    </>
  );
}
