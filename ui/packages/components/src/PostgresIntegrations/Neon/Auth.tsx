import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Link } from '@inngest/components/Link';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';
import { parseConnectionString } from '@inngest/components/PostgresIntegrations/utils';
import { cn } from '@inngest/components/utils/classNames';

const defaultIntro = `
Inngest needs to be authorized with your postgres credentials to set up replication slots,
publications, and a new user that subscribes to updates. Note that your admin credentials
will not be stored and are only used for setup.
`;

const defaultCredsLink = 'https://neon.tech/docs/connect/connect-securely';

export default function NeonAuth({
  onSuccess,
  savedCredentials,
  verifyCredentials,
  integration = 'neon', // eg. "neon", "supabase" - the integration name
  intro = defaultIntro,
  credsLink = defaultCredsLink,
  nextStep = IntegrationSteps.FormatWal,
}: {
  onSuccess: (value: string) => void;
  savedCredentials?: string;
  integration: string;
  intro: string;
  credsLink: string;
  nextStep: IntegrationSteps;
  verifyCredentials: (variables: {
    adminConn: string;
    engine: string;
    name: string;
    replicaConn?: string;
  }) => Promise<{ success: boolean; error: string }>;
}) {
  const [inputValue, setInputValue] = useState(savedCredentials || '');
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState<string>();
  const [isVerified, setIsVerified] = useState(!!savedCredentials);

  useEffect(() => {
    if (savedCredentials) {
      setInputValue(savedCredentials);
      setIsVerified(true);
    }
  }, [savedCredentials]);

  const handleVerify = async () => {
    setIsVerifying(true);
    setError(undefined);

    const parsedInput = parseConnectionString(integration, inputValue);

    if (!parsedInput) {
      setError('Invalid connection string format. Please check your input.');
      setIsVerifying(false);
      return;
    }

    try {
      const { success, error } = await verifyCredentials(parsedInput);
      if (success) {
        setIsVerified(true);
        onSuccess(inputValue);
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
      <p className="pb-1 text-sm">{intro}</p>
      <Link size="small" href={credsLink} target="_blank">
        Learn more about postgres credentials.
      </Link>
      <form
        className={cn('pt-6', isVerified || error ? 'pb-2' : 'pb-8')}
        onSubmit={(e) => e.preventDefault()}
      >
        <label className="pb-2 text-sm">
          Please paste your admin postgres credentials in the field below to continue:
          <div className="flex items-start justify-between gap-1 pt-2">
            <div className="w-full">
              <Input
                placeholder="eg: postgresql://user:6sFm9owfZqSk@your-db-host.db"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                error={error}
              />
            </div>
            <Button label="Verify" onClick={handleVerify} loading={isVerifying} />
          </div>
        </label>
        {isVerified && (
          <p className="text-success mt-1 text-sm">Credentials verified successfully!</p>
        )}
      </form>
      <Button
        label="Next"
        href={`/settings/integrations/${integration}/${nextStep}`}
        disabled={!isVerified}
      />
    </>
  );
}
