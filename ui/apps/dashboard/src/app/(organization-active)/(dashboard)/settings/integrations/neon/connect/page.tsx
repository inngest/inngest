'use client';

import { useRouter } from 'next/navigation';
import ConnectPage from '@inngest/components/PostgresIntegrations/ConnectPage';
import { neonConnectContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import { STEPS_ORDER } from '@inngest/components/PostgresIntegrations/types';

import { pathCreator } from '@/utils/urls';

export default function NeonConnect() {
  const router = useRouter();
  const firstStep = STEPS_ORDER[0]!;

  return (
    <ConnectPage
      content={neonConnectContent}
      onStartInstallation={() => {
        router.push(pathCreator.pgIntegrationStep({ integration: 'neon', step: firstStep }));
      }}
    />
  );
}
