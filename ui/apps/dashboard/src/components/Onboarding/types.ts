import { type MenuStepContent } from '@inngest/components/Steps/StepsMenu';

export enum OnboardingSteps {
  CreateApp = 'create-app',
  DeployApp = 'deploy-app',
  SyncApp = 'sync-app',
  InvokeFn = 'invoke-fn',
}

export type OnboardingStep = {
  name: OnboardingSteps;
  stepNumber: ArrayLengthRange<typeof steps>;
  isFinalStep: boolean;
};

export const steps: OnboardingStep[] = [
  {
    name: OnboardingSteps.CreateApp,
    stepNumber: 1,
    isFinalStep: false,
  },
  {
    name: OnboardingSteps.DeployApp,
    stepNumber: 2,
    isFinalStep: false,
  },
  {
    name: OnboardingSteps.SyncApp,
    stepNumber: 3,
    isFinalStep: false,
  },
  {
    name: OnboardingSteps.InvokeFn,
    stepNumber: 4,
    isFinalStep: true,
  },
];

export const STEPS_ORDER: OnboardingSteps[] = [
  OnboardingSteps.CreateApp,
  OnboardingSteps.DeployApp,
  OnboardingSteps.SyncApp,
  OnboardingSteps.InvokeFn,
];

type ArrayLengthRange<T extends readonly any[]> = Extract<
  keyof { [K in 0 | T['length']]: K },
  number
>;

// Type representing the possible number of completed steps (0 to 4)
export type TotalStepsCompleted = ArrayLengthRange<typeof STEPS_ORDER>;

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
