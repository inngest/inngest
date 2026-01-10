import { neonMenuStepContent } from '@inngest/components/PostgresIntegrations/Neon/neonContent';
import {
  IntegrationSteps,
  STEPS_ORDER,
  isValidStep,
} from '@inngest/components/PostgresIntegrations/types';
import StepsMenu from '@inngest/components/Steps/StepsMenu';
import { RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { useSteps } from './Context';

import Menu from '@inngest/components/PostgresIntegrations/StepsMenu';
import { useNavigate } from '@tanstack/react-router';

export default function NeonStepsMenu({
  step,
  steps = STEPS_ORDER,
}: {
  step: string;
  steps?: IntegrationSteps[];
}) {
  const navigate = useNavigate();
  const { stepsCompleted } = useSteps();
  if (!isValidStep(step)) {
    navigate({
      to: pathCreator.pgIntegrationStep({ integration: 'supabase' }),
    });
    return;
  }

  return (
    <Menu
      stepsCompleted={stepsCompleted}
      activeStep={step}
      content={neonMenuStepContent}
      links={links}
      steps={steps}
      pathname={pathCreator.pgIntegrationStep({ integration: 'supabase' })}
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
