'use client';

import dynamic from 'next/dynamic';
import { useRouter } from 'next/navigation';
import { neonMenuStepContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import { STEPS_ORDER, isValidStep } from '@inngest/components/PostgresIntegrations/types';
import StepsMenu from '@inngest/components/Steps/StepsMenu';
import { RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { useSteps } from './Context';

const Menu = dynamic(() => import('@inngest/components/PostgresIntegrations/StepsMenu'), {
  ssr: false,
});

export default function NeonStepsMenu({ step }: { step: string }) {
  const router = useRouter();
  const { stepsCompleted } = useSteps();
  if (!isValidStep(step)) {
    router.push(pathCreator.neonIntegrationStep({}));
    return;
  }

  return (
    <Menu
      stepsCompleted={stepsCompleted}
      activeStep={step}
      content={neonMenuStepContent}
      links={links}
      steps={STEPS_ORDER}
      pathname={pathCreator.neonIntegrationStep({})}
    />
  );
}

const links = (
  <StepsMenu.Link
    iconBefore={<RiExternalLinkLine className="h-4 w-4" />}
    href="https://www.inngest.com/docs"
  >
    See documentation
  </StepsMenu.Link>
);
