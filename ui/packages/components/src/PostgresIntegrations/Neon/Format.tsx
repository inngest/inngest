import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { InlineCode } from '@inngest/components/Code';
import { Link } from '@inngest/components/Link';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/utils';

export default function NeonFormat({
  onSuccess,
  savedCredentials,
  verifyLogicalReplication,
  handleLostCredentials,
  integration,
}: {
  onSuccess: () => void;
  handleLostCredentials: () => void;
  savedCredentials?: string;
  integration: string;
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
        Enabling logical replication modifies the Postgres <InlineCode>wal_level</InlineCode>{' '}
        configuration parameter, changing it from <InlineCode>replica</InlineCode> to{' '}
        <InlineCode>logical</InlineCode> for all databases in your Neon project. Once the{' '}
        <InlineCode>wal_level</InlineCode> setting is changed to <InlineCode>logical</InlineCode>,
        it cannot be reverted. Enabling logical replication also restarts all computes in your Neon
        project, meaning active connections will be dropped and have to reconnect.
      </p>
      <Link
        size="small"
        href="https://neon.tech/docs/guides/logical-replication-concepts#write-ahead-log-wal"
      >
        Learn more about WAL level
      </Link>

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
        <Button
          label="Next"
          href={`/settings/integrations/${integration}/${IntegrationSteps.ConnectDb}`}
        />
      ) : (
        <Button
          label="Verify logical replication is enabled"
          onClick={handleVerify}
          loading={isVerifying}
        />
      )}
      {error && <p className="text-error mt-4 text-sm">{error}</p>}
    </>
  );
}
