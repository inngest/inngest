import { type OnboardingWidgetContent } from './types';

export const onboardingWidgetContent: OnboardingWidgetContent = {
  step: {
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
      title: 'Almost there',
      description: "Let's get your system up and running on Inngest.",
      eta: 'Est. 3 mins remaining',
    },
    4: {
      title: 'Well done',
      description: 'You can now explore the full capabilities of Inngest.',
      cta: 'View our starter plans',
    },
  },
  tooltip: {
    close: "Close this widget. Reopen from the 'Help & Feedback' menu, if needed.",
  },
};
