'use client';

import { CodeKey } from '@inngest/components/CodeKey';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';

type DeploySigningKeyProps = {
  className?: string;
};

export default function DeploySigningKey({ className }: DeploySigningKeyProps) {
  const environment = useEnvironment();
  const signingKey = environment.webhookSigningKey || '...';
  const maskedSigningKey = environment.webhookSigningKey
    ? environment.webhookSigningKey.replace(/signkey-(prod|test)-.+/, 'signkey-$1')
    : '...';

  return (
    <span className={className}>
      <CodeKey fullKey={signingKey} maskedKey={maskedSigningKey} />
    </span>
  );
}
