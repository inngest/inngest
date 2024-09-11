'use client';

import { usePathname } from 'next/navigation';
import StepsMenu from '@inngest/components/Steps/StepsMenu';
import { RiDiscordLine, RiExternalLinkLine, RiMailLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { type OnboardingStepsArray } from '../Onboarding/types';
import { onboardingMenuStepContent } from './content';
import useOnboardingStep from './useOnboardingStep';

export default function Menu({ envSlug }: { envSlug: string }) {
  const { lastCompletedStep } = useOnboardingStep();
  const pathname = usePathname();

  const activeStep = pathname.split('/').pop() || '1';
  const stepNumbers: OnboardingStepsArray = [1, 2, 3, 4];

  return (
    <StepsMenu title={onboardingMenuStepContent.title} links={links}>
      {stepNumbers.map((stepNumber) => {
        const isCompleted = stepNumber <= lastCompletedStep;
        const isActive = activeStep === stepNumber.toString();
        const stepContent = onboardingMenuStepContent.step[stepNumber];
        const url = pathCreator.onboardingSteps({ envSlug: envSlug, step: stepNumber });
        return (
          <StepsMenu.MenuItem
            key={stepNumber}
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
