import { type MenuStepContent } from '@inngest/components/Steps/StepsMenu';

export type OnboardingSteps = 1 | 2 | 3 | 4;
// For localStorage
export type OnboardingStepsString = `${OnboardingSteps}`;

type OnboardingWidgetStepContent = {
  title: string;
  description: string;
  cta?: string;
  eta?: string;
};

export type OnboardingWidgetContent = {
  step: {
    [K in 0 | OnboardingSteps]: OnboardingWidgetStepContent;
  };
  tooltip: {
    close: string;
  };
};

export type OnboardingMenuContent = {
  step: {
    [K in OnboardingSteps]: MenuStepContent;
  };
  title: string;
};
