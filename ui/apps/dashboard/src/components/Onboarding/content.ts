import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { RiLoopRightLine, RiMagicLine, RiSendPlaneLine } from '@remixicon/react';

import { type OnboardingMenuContent, type OnboardingWidgetContent } from './types';

export const onboardingWidgetContent: OnboardingWidgetContent = {
  step: {
    'create-app': {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 10 mins remaining',
    },
    'deploy-app': {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 10 mins remaining',
    },
    'sync-app': {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 7 mins remaining',
    },
    'invoke-fn': {
      title: 'Almost there!',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 3 mins remaining',
    },
    success: {
      title: 'Well done!',
      description: 'You can now explore the full capabilities of Inngest.',
      cta: 'View our starter plans',
    },
  },
  tooltip: {
    close: "Close this widget. Reopen from the 'Help & Feedback' menu, if needed.",
  },
};

export const onboardingMenuStepContent: OnboardingMenuContent = {
  title: 'Explore onboarding guide',
  step: {
    'create-app': {
      title: 'Create Inngest app',
      description: 'Start building in local development',
      icon: AppsIcon,
    },
    'deploy-app': {
      title: 'Deploy Inngest app',
      description: 'Host your app on any platform or infra',
      icon: RiSendPlaneLine,
    },
    'sync-app': {
      title: 'Sync app to Inngest',
      description: 'Tell Inngest where your app is running',
      icon: RiLoopRightLine,
    },
    'invoke-fn': {
      title: 'Invoke your function',
      description: 'Trigger and monitor your first function',
      icon: RiMagicLine,
    },
  },
} as const;
