'use client';

import { useRouter } from 'next/navigation';
import ConnectPage from '@inngest/components/PostgresIntegrations/ConnectPage';
import { connectContent } from '@inngest/components/PostgresIntegrations/Supabase/supabaseContent';
import { STEPS_ORDER } from '@inngest/components/PostgresIntegrations/types';

import { pathCreator } from '@/utils/urls';

export default function Connect() {
  const router = useRouter();
  const firstStep = STEPS_ORDER[0]!;
  return (
    <ConnectPage
      content={connectContent}
      onStartInstallation={() => {
        router.push(pathCreator.pgIntegrationStep({ integration: 'supabase', step: firstStep }));
      }}
    />
  );
}
