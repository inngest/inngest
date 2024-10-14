'use client';

import StepsMenu from '@inngest/components/Steps/StepsMenu';
import { RiDiscordLine, RiExternalLinkLine, RiMailLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { STEPS_ORDER, isValidStep } from '../Onboarding/types';
import { onboardingMenuStepContent } from './content';
import useOnboardingStep from './useOnboardingStep';

export default function Menu({ envSlug, step }: { envSlug: string; step: string }) {
  const { lastCompletedStep } = useOnboardingStep();

  return (
    <StepsMenu title={onboardingMenuStepContent.title} links={links}>
      {STEPS_ORDER.map((stepKey) => {
        if (!isValidStep(stepKey)) {
          return 'error';
        }
        const isCompleted = lastCompletedStep
          ? STEPS_ORDER.indexOf(stepKey) <= STEPS_ORDER.indexOf(lastCompletedStep)
          : false;
        const isActive = step === stepKey;
        const stepContent = onboardingMenuStepContent.step[stepKey];
        const url = pathCreator.onboardingSteps({ envSlug: envSlug, step: stepKey });
        return (
          <StepsMenu.MenuItem
            key={stepKey}
            stepContent={stepContent}
            isCompleted={isCompleted}
            isActive={isActive}
            url={url}
          />
        );
      })}
    </StepsMenu>
  );
}

const links = (
  <>
    <StepsMenu.Link
      iconBefore={<RiExternalLinkLine className="h-4 w-4" />}
      href="https://www.inngest.com/docs"
    >
      See documentation
    </StepsMenu.Link>
    <StepsMenu.Link
      iconBefore={<RiDiscordLine className="h-4 w-4" />}
      href="https://www.inngest.com/discord"
    >
      Join discord community
    </StepsMenu.Link>
    <StepsMenu.Link iconBefore={<RiMailLine className="h-4 w-4" />} href={pathCreator.support()}>
      Request a demo
    </StepsMenu.Link>
  </>
);
