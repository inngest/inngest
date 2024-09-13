import StepsPageHeader from '@inngest/components/Steps/StepsPageHeader';

import { onboardingMenuStepContent } from '@/components/Onboarding/content';
import { steps } from '@/components/Onboarding/types';

export default function Page() {
  const totalSteps = steps.length;
  const title = onboardingMenuStepContent.step[2].title;
  return (
    <div>
      <StepsPageHeader currentStep={2} totalSteps={totalSteps} title={title} />
    </div>
  );
}
