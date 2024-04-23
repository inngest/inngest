'use client';

import { Alert } from '@inngest/components/Alert';
import { Card } from '@inngest/components/Card';
import { Link } from '@inngest/components/Link';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import LoadingIcon from '@/icons/LoadingIcon';
import { CreateSigningKeyButton } from './CreateSigningKeyButton';
import { InlineCode } from './InlineCode';
import { RotateSigningKeyButton } from './RotateSigningKeyButton';
import { SigningKey } from './SigningKey';
import { useSigningKeys } from './useSigningKeys';

export default function Page() {
  const env = useEnvironment();

  const { data, error, isLoading } = useSigningKeys({ envID: env.id });
  if (error) {
    throw error;
  }
  if (isLoading && !data) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const activeKeys = data.filter((key) => key.isActive);
  if (!activeKeys[0]) {
    // Unreachable
    throw new Error('No active key found');
  }
  if (activeKeys.length > 1) {
    // Unreachable
    throw new Error('More than one active key found');
  }
  const activeKey = activeKeys[0];
  const inactiveKeys = data.filter((key) => !key.isActive);

  return (
    <div className="my-8 flex items-center justify-center">
      <div className="w-full max-w-[800px] divide-y">
        <div className="mb-8">
          <h1 className="mb-2 text-2xl">Signing keys</h1>

          <p className="mb-8 text-sm text-slate-500">
            Signing keys are secrets used for secure communication between Inngest and your apps.
            <Link
              internalNavigation={false}
              href="https://www.inngest.com/docs/security#signing-keys-and-sdk-security"
            >
              Learn More
            </Link>
          </p>

          <SigningKey signingKey={activeKey} />

          {inactiveKeys.map((signingKey) => {
            return <SigningKey key={signingKey.id} signingKey={signingKey} />;
          })}

          <CreateSigningKeyButton disabled={data.length > 1} envID={env.id} />
        </div>

        <div>
          <h2 className="mb-2 mt-4 text-xl">Rotation</h2>

          <div className="mb-8 text-sm text-slate-500">
            Create a new signing key and update environment variables in your app: set{' '}
            <InlineCode value="INNGEST_SIGNING_KEY" /> to the value of the{' '}
            <span className="font-bold">new key</span> and{' '}
            <InlineCode value="INNGEST_SIGNING_KEY_FALLBACK" /> to the value of the{' '}
            <span className="font-bold">current key</span>. Deploy your apps and then click the{' '}
            <span className="font-bold">Rotate key</span> button.
          </div>

          <Card>
            <Card.Content className="p-4">
              <div className="mb-4 flex items-center">
                <div className="grow">
                  <p className="mb-2 font-medium">Rotate key</p>

                  <p className="text-sm text-slate-500">
                    This action replaces the <span className="font-bold">current key</span> with the{' '}
                    <span className="font-bold">new key</span>, permanently deleting the current
                    key.
                  </p>
                </div>

                <RotateSigningKeyButton disabled={inactiveKeys.length === 0} envID={env.id} />
              </div>

              <Alert severity="warning">
                <p className="mb-2">
                  Rotation may cause downtime if your SDK does not meet the minimum version.
                  <Link
                    internalNavigation={false}
                    href="https://www.inngest.com/docs/security#rotation"
                  >
                    Learn More
                  </Link>
                </p>
              </Alert>
            </Card.Content>
          </Card>
        </div>
      </div>
    </div>
  );
}
