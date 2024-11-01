'use client';

import { useRouter } from 'next/navigation';
import Auth from '@inngest/components/PostgresIntegrations/Neon/Auth';
import Connect from '@inngest/components/PostgresIntegrations/Neon/Connect';
import { IntegrationSteps, STEPS_ORDER } from '@inngest/components/PostgresIntegrations/types';
import { toast } from 'sonner';

import { useSteps } from '@/components/PostgresIntegration/Context';
import { pathCreator } from '@/utils/urls';
import { verifyAutoSetup, verifyCredentials } from './actions';

export default function Step({ params: { step } }: { params: { step: string } }) {
  const { setStepsCompleted, credentials, setCredentials } = useSteps();
  const router = useRouter();
  const firstStep = STEPS_ORDER[0]!;

  function handleLostCredentials() {
    toast.error('Lost credentials. Going back to the first step.');
    router.push(pathCreator.pgIntegrationStep({ integration: 'supabase', step: firstStep }));
  }

  if (step === IntegrationSteps.Authorize) {
    return (
      <Auth
        savedCredentials={credentials}
        integration="supabase"
        onSuccess={(value) => {
          setCredentials(value);
          setStepsCompleted(IntegrationSteps.Authorize);
        }}
        // @ts-ignore for now
        verifyCredentials={verifyCredentials}
        nextStep={IntegrationSteps.ConnectDb}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <Connect
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
        }}
        integration="supabase"
        // @ts-ignore for now
        verifyAutoSetup={verifyAutoSetup}
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  }

  return <div>Page Content</div>;
}
