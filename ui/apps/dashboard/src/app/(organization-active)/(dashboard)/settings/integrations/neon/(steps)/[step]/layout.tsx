'use client';

import Menu from '@inngest/components/PostgresIntegrations/StepsMenu';
import { neonMenuStepContent } from '@inngest/components/PostgresIntegrations/neonContent';
import {
  STEPS_ORDER,
  isValidStep,
  type IntegrationSteps,
} from '@inngest/components/PostgresIntegrations/types';
import StepsMenu from '@inngest/components/Steps/StepsMenu';
import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';
import { RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';

export default function Layout({
  children,
  params: { step },
}: React.PropsWithChildren<{ params: { step: string } }>) {
  if (!isValidStep(step)) {
    // To Do: Handle invalid step, e.g., redirect to first step or show an error
    return <div>Invalid step.</div>;
  }

  const currentStep = STEPS_ORDER.indexOf(step as IntegrationSteps);
  const stepContent = neonMenuStepContent.step[step as IntegrationSteps];

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
        lastCompletedStep={'authorize'}
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
