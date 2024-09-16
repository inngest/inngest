import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { RiLoopRightLine, RiMagicLine, RiSendPlaneLine } from '@remixicon/react';

import { type OnboardingMenuContent, type OnboardingWidgetContent } from './types';

export const onboardingWidgetContent: OnboardingWidgetContent = {
  step: {
    0: {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 10 mins remaining',
    },
    1: {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 10 mins remaining',
    },
    2: {
      title: 'Getting started',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 7 mins remaining',
    },
    3: {
      title: 'Almost there!',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 3 mins remaining',
    },
    4: {
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
    1: {
      title: 'Create Inngest app',
      description: 'Start building in local development',
      icon: AppsIcon,
    },
    2: {
      title: 'Deploy Inngest app',
      description: 'Host your app on any platform or infra',
      icon: RiSendPlaneLine,
    },
    3: {
      title: 'Sync app to Inngest',
      description: 'Tell Inngest where your app is running',
      icon: RiLoopRightLine,
    },
    4: {
      title: 'Invoke your function',
      description: 'Trigger and monitor your first function',
      icon: RiMagicLine,
    },
  },
} as const;
