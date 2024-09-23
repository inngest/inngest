'use client';

import NeonAuth from '@inngest/components/PostgresIntegrations/Neon/Auth';
import NeonConnect from '@inngest/components/PostgresIntegrations/Neon/Connect';
import NeonFormat from '@inngest/components/PostgresIntegrations/Neon/Format';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

import { useSteps } from '@/components/PostgresIntegration/Context';
import { verifyCredentials } from './actions';

export default function NeonStep({ params: { step } }: { params: { step: string } }) {
  const { setStepsCompleted, credentials, setCredentials } = useSteps();

  if (step === IntegrationSteps.Authorize) {
    return (
      <NeonAuth
        savedCredentials={credentials}
        onSuccess={(value) => {
          setCredentials(value);
          setStepsCompleted(IntegrationSteps.Authorize);
        }}
        verifyCredentials={verifyCredentials}
      />
    );
  } else if (step === IntegrationSteps.FormatWal) {
    return (
      <NeonFormat
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.FormatWal);
        }}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <NeonConnect
        onSuccess={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
        }}
      />
    );
  }

  return <div>Page Content</div>;
}
