'use client';

import { useRouter } from 'next/navigation';
import NeonAuth from '@inngest/components/PostgresIntegrations/Neon/Auth';
import NeonConnect from '@inngest/components/PostgresIntegrations/Neon/Connect';
import NeonFormat from '@inngest/components/PostgresIntegrations/Neon/Format';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

import { useSteps } from '@/components/PostgresIntegration/Context';
import { pathCreator } from '@/utils/urls';

export default function NeonStep({ params: { step } }: { params: { step: string } }) {
  const router = useRouter();
  const { setStepsCompleted } = useSteps();

  if (step === IntegrationSteps.Authorize) {
    return (
      <NeonAuth
        next={() => {
          setStepsCompleted(IntegrationSteps.Authorize);
          router.push(pathCreator.neonIntegrationStep({ step: IntegrationSteps.FormatWal }));
        }}
      />
    );
  } else if (step === IntegrationSteps.FormatWal) {
    return (
      <NeonFormat
        next={() => {
          setStepsCompleted(IntegrationSteps.FormatWal);
          router.push(pathCreator.neonIntegrationStep({ step: IntegrationSteps.ConnectDb }));
        }}
      />
    );
  } else if (step === IntegrationSteps.ConnectDb) {
    return (
      <NeonConnect
        next={() => {
          setStepsCompleted(IntegrationSteps.ConnectDb);
          router.push(pathCreator.neonIntegrationStep({ step: IntegrationSteps.ConnectDb }));
        }}
      />
    );
  }

  return <div>Page Content</div>;
}
