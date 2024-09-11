import { type MenuStepContent } from '@inngest/components/Steps/StepsMenu';

export type OnboardingSteps = 1 | 2 | 3 | 4;
export type OnboardingStepsArray = OnboardingSteps[];
// For localStorage
export type OnboardingStepsString = `${OnboardingSteps}`;

export type OnboardingStepsCompleted = 0 | OnboardingSteps;

type OnboardingWidgetStepContent = {
  title: string;
  description: string;
  cta?: string;
  eta?: string;
};

export type OnboardingWidgetContent = {
  step: {
    [K in OnboardingStepsCompleted]: OnboardingWidgetStepContent;
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
