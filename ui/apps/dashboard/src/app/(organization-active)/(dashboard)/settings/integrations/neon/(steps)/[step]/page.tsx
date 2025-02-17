'use client';

import { useRouter } from 'next/navigation';
import NeonAuth from '@inngest/components/PostgresIntegrations/Neon/Auth';
import NeonConnect from '@inngest/components/PostgresIntegrations/Neon/Connect';
import NeonFormat from '@inngest/components/PostgresIntegrations/Neon/Format';
import { IntegrationSteps, STEPS_ORDER } from '@inngest/components/PostgresIntegrations/types';
import { toast } from 'sonner';

import { useSteps } from '@/components/PostgresIntegration/Context';
import { pathCreator } from '@/utils/urls';
import { verifyAutoSetup, verifyCredentials, verifyLogicalReplication } from './actions';

export default function NeonStep({ params: { step } }: { params: { step: string } }) {
  const { setStepsCompleted, credentials, setCredentials } = useSteps();
  const router = useRouter();
  const firstStep = STEPS_ORDER[0]!;

  function handleLostCredentials() {
    toast.error('Lost credentials. Going back to the first step.');
    router.push(pathCreator.pgIntegrationStep({ integration: 'neon', step: firstStep }));
  }

  if (step === IntegrationSteps.Authorize) {
    return (
      <NeonAuth
        savedCredentials={credentials}
        onSuccess={(value) => {
          setCredentials(value);
          setStepsCompleted(IntegrationSteps.Authorize);
        }}
        integration="neon"
        // @ts-ignore for now
        verifyCredentials={verifyCredentials}
      />
    );
  } else if (step === IntegrationSteps.FormatWal) {
    return (
      <NeonFormat
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.FormatWal);
        }}
        integration="neon"
        // @ts-ignore for now
        verifyLogicalReplication={verifyLogicalReplication}
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <NeonConnect
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
        }}
        integration="neon"
        // @ts-ignore for now
        verifyAutoSetup={verifyAutoSetup}
        savedCredentials={credentials}
        handleLostCredentials={handleLostCredentials}
      />
    );
  }

  return <div>Page Content</div>;
}
