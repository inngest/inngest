'use client';

import { useRouter } from 'next/navigation';
import NeonAuth from '@inngest/components/PostgresIntegrations/Neon/Auth';
import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

import { pathCreator } from '@/utils/urls';
import { useSteps } from '../Context';

export default function NeonStep({ params: { step } }: { params: { step: string } }) {
  const router = useRouter();
  const { setStepsCompleted } = useSteps();

  if (step === IntegrationSteps.Authorize) {
    return (
      <NeonAuth
        next={() => {
          setStepsCompleted([IntegrationSteps.Authorize]);
          router.push(pathCreator.neonIntegrationStep({ step: IntegrationSteps.FormatWal }));
        }}
      />
    );
  } else if (step === IntegrationSteps.FormatWal) {
    return <div>Page for Format</div>;
  } else if (step === IntegrationSteps.ConnectDb) {
    return <div>Page For Connect</div>;
  }

  return <div>Page Content</div>;
}
