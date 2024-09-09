import { type RemixiconComponentType } from '@remixicon/react';

export type OnboardingSteps = 1 | 2 | 3 | 4;
type OnboardingWidgetStepContent = {
  title: string;
  description: string;
  cta?: string;
  eta?: string;
};

export type OnboardingWidgetContent = {
  step: {
    [K in OnboardingSteps]: OnboardingWidgetStepContent;
  };
  tooltip: {
    close: string;
  };
};

type OnboardingMenuStepContent = {
  title: string;
  description: string;
  icon: RemixiconComponentType | React.ReactNode;
};

export type OnboardingMenuContent = {
  step: {
    [K in OnboardingSteps]: OnboardingMenuStepContent;
  };
  title: string;
};
