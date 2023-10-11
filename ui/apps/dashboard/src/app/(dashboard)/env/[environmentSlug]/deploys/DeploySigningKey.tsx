'use client';

import SecretKey from '@/components/Secrets/SecretKey';
import { useEnvironment } from '@/queries';

type DeploySigningKeyProps = {
  environmentSlug: string;
  context?: 'dark' | 'light';
};

export default function DeploySigningKey({
  environmentSlug,
  context = 'light',
}: DeploySigningKeyProps) {
  const [{ data: environment }] = useEnvironment({ environmentSlug });
  const signingKey = environment?.webhookSigningKey || '...';
  const maskedSigningKey = environment?.webhookSigningKey
    ? environment.webhookSigningKey.replace(
        /signkey-(prod|test)-.+/,
        'signkey-$1-<click-to-reveal>'
      )
    : '...';

  return <SecretKey value={signingKey} masked={maskedSigningKey} context={context} />;
}
