'use client';

import dynamic from 'next/dynamic';
import { useRouter } from 'next/navigation';
import { neonMenuStepContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import { STEPS_ORDER, isValidStep } from '@inngest/components/PostgresIntegrations/types';
import StepsMenu from '@inngest/components/Steps/StepsMenu';
import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';
import { RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { useSteps } from '../Context';

const Menu = dynamic(() => import('@inngest/components/PostgresIntegrations/StepsMenu'), {
  ssr: false,
});

export default function Layout({
  children,
  params: { step },
}: React.PropsWithChildren<{ params: { step: string } }>) {
  const router = useRouter();
  const { stepsCompleted } = useSteps();
  if (!isValidStep(step)) {
    router.push(pathCreator.neonIntegrationStep({}));
    return;
  }
  const currentStep = STEPS_ORDER.indexOf(step);
  const stepContent = neonMenuStepContent.step[step];

  return (
    <div className="my-12 grid grid-cols-3">
      <main className="col-span-2 mx-20">
        <StepsPageHeader
          currentStep={currentStep + 1}
          totalSteps={STEPS_ORDER.length}
          title={stepContent.title}
        />
        {children}
      </main>
      <Menu
        stepsCompleted={stepsCompleted}
        activeStep={step}
        content={neonMenuStepContent}
        links={links}
        steps={STEPS_ORDER}
        pathname={pathCreator.neonIntegrationStep({})}
      />
    </div>
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
