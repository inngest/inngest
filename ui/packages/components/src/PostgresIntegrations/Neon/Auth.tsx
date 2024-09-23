import { useEffect, useState } from 'react';
import { NewButton } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { NewLink } from '@inngest/components/Link';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';
import { cn } from '@inngest/components/utils/classNames';

const verifyCredentials = async (credentials: string): Promise<boolean> => {
  // TO DO: Replace with actual API call in production.
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(credentials.startsWith('postgresql://'));
    }, 1000);
  });
};

export default function NeonAuth({
  onSuccess,
  savedCredentials,
}: {
  onSuccess: (value: string) => void;
  savedCredentials?: string;
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
    try {
      const success = await verifyCredentials(inputValue);
      if (success) {
        setIsVerified(true);
        onSuccess(inputValue);
      } else {
        setError(
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
        Inngest needs to be authorized with your postgres credentials to set up replication slots,
        publications, and a new user that subscribes to updates. Note that your admin credentials
        will not be stored and are only used for setup.
      </p>
      <NewLink size="small" href="https://neon.tech/docs/connect/connect-securely">
        Learn more about postgres credentials
      </NewLink>
      <form
        className={cn('pt-6', isVerified || error ? 'pb-2' : 'pb-8')}
        onSubmit={(e) => e.preventDefault()}
      >
        <label className="pb-2 text-sm">
          Please paste your admin postgres credentials in the field below to continue.
        </label>
        <div className="flex items-start justify-between gap-1">
          <div className="w-full">
            <Input
              placeholder="eg: postgresql://neondb_owner:6sFm9owfZqSk@a5hly6e1.useast-2.aws.tech"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              error={error}
            />
          </div>
          <NewButton label="Verify" onClick={handleVerify} loading={isVerifying} />
        </div>
        {isVerified && (
          <p className="text-success mt-1 text-sm">Credentials verified successfully!</p>
        )}
      </form>
      <NewButton
        label="Next"
        href={`/settings/integrations/neon/${IntegrationSteps.FormatWal}`}
        disabled={!isVerified}
      />
    </>
  );
}
