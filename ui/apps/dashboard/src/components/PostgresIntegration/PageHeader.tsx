import { useNavigate } from "@tanstack/react-router";
import { neonMenuStepContent } from "@inngest/components/PostgresIntegrations/Neon/neonContent";
import {
  IntegrationSteps,
  STEPS_ORDER,
  isValidStep,
} from "@inngest/components/PostgresIntegrations/types";
import StepsPageHeader from "@inngest/components/Steps/StepsPageHeader";
import { pathCreator } from "@/utils/urls";

export default function PageHeader({
  step,
  integration,
  steps = STEPS_ORDER,
}: {
  step: string;
  integration: string;
  steps?: IntegrationSteps[];
}) {
  const navigate = useNavigate();

  if (!isValidStep(step)) {
    navigate({
      to: pathCreator.pgIntegrationStep({ integration }),
      params: { step: STEPS_ORDER[0]! },
    });
    return;
  }
  const currentStep = steps.indexOf(step);
  const stepContent = neonMenuStepContent.step[step];

  return (
    <StepsPageHeader
      currentStep={currentStep + 1}
      totalSteps={steps.length}
      title={stepContent.title}
    />
  );
}
