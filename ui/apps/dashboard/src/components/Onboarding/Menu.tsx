'use client';

import StepsMenu from '@inngest/components/Steps/StepsMenu';
import { RiDiscordLine, RiExternalLinkLine, RiMailLine } from '@remixicon/react';

import { WEBSITE_CONTACT_URL, pathCreator } from '@/utils/urls';
import { isValidStep, steps } from '../Onboarding/types';
import { onboardingMenuStepContent } from './content';
import useOnboardingStep from './useOnboardingStep';

export default function Menu({ envSlug, stepName }: { envSlug: string; stepName: string }) {
  const { completedSteps } = useOnboardingStep();

  return (
    <StepsMenu title={onboardingMenuStepContent.title} links={links}>
      {steps.map((stepObj) => {
        const { name, stepNumber } = stepObj;

        if (!isValidStep(stepName)) {
          return 'error';
        }

        const isCompleted = completedSteps.some((step) => step.stepNumber === stepNumber);

        const isActive = stepName === name;

        const stepContent = onboardingMenuStepContent.step[name];
        const url = pathCreator.onboardingSteps({ envSlug: envSlug, step: name });

        return (
          <StepsMenu.MenuItem
            key={name}
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
      href="https://www.inngest.com/docs?ref=app-onboarding-menu"
      target="_blank"
    >
      See documentation
    </StepsMenu.Link>
    <StepsMenu.Link
      iconBefore={<RiDiscordLine className="h-4 w-4" />}
      href="https://www.inngest.com/discord?ref=app-onboarding-menu"
      target="_blank"
    >
      Join discord community
    </StepsMenu.Link>
    <StepsMenu.Link
      iconBefore={<RiMailLine className="h-4 w-4" />}
      href={WEBSITE_CONTACT_URL + '?ref=app-onboarding-menu'}
      target="_blank"
    >
      Request a demo
    </StepsMenu.Link>
  </>
);
