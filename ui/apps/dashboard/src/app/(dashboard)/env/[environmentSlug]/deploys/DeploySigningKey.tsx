'use client';

import { CodeKey } from '@inngest/components/CodeKey';

import { useEnvironment } from '@/queries';

type DeploySigningKeyProps = {
  environmentSlug: string;
  className?: string;
};

export default function DeploySigningKey({ environmentSlug, className }: DeploySigningKeyProps) {
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const signingKey = environment?.webhookSigningKey || '...';
  const maskedSigningKey = environment?.webhookSigningKey
    ? environment.webhookSigningKey.replace(
        /signkey-(prod|test)-.+/,
        'signkey-$1-<click-to-reveal>'
      )
    : '...';

  return (
    <span className={className}>
      <CodeKey fullKey={signingKey} maskedKey={maskedSigningKey} />
    </span>
  );
}
