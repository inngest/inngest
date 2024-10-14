import { type MenuStepContent } from '@inngest/components/Steps/StepsMenu';

export enum OnboardingSteps {
  CreateApp = 'create-app',
  DeployApp = 'deploy-app',
  SyncApp = 'sync-app',
  InvokeFn = 'invoke-fn',
}

export const STEPS_ORDER: OnboardingSteps[] = [
  OnboardingSteps.CreateApp,
  OnboardingSteps.DeployApp,
  OnboardingSteps.SyncApp,
  OnboardingSteps.InvokeFn,
];

export function isValidStep(step: string): step is OnboardingSteps {
  return STEPS_ORDER.includes(step as OnboardingSteps);
}

export type OnboardingStepsCompleted = OnboardingSteps[] | [];

type OnboardingWidgetStepContent = {
  title: string;
  description: string;
  cta?: string;
  eta?: string;
};

export type OnboardingWidgetContent = {
  step: {
    [K in OnboardingSteps | 'success']: OnboardingWidgetStepContent;
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
