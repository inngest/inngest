'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { NewLink } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiCheckboxCircleFill,
  RiDiscordLine,
  RiExternalLinkLine,
  RiMailLine,
} from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { type OnboardingSteps } from '../Onboarding/types';
import { onboardingMenuStepContent } from './content';
import { type OnboardingMenuStepContent as OnboardingMenuStepContentProps } from './types';
import useOnboardingStep from './useOnboardingStep';

export default function Menu({ envSlug }: { envSlug: string }) {
  const { lastCompletedStep } = useOnboardingStep();
  const pathname = usePathname();

  const activeStep = pathname.split('/').pop() || '1';
  const stepNumbers: OnboardingSteps[] = [1, 2, 3, 4];

  return (
    <div className="mr-12 flex flex-col">
      <nav className="mb-12">
        <h3 className="text-muted text-xs font-medium uppercase">
          {onboardingMenuStepContent.title}
        </h3>
        <ul className="my-2">
          {stepNumbers.map((stepNumber) => {
            const isCompleted = stepNumber <= lastCompletedStep;
            const isActive = activeStep === stepNumber.toString();
            const stepContent = onboardingMenuStepContent.step[stepNumber];
            const url = pathCreator.onboardingSteps({ envSlug: envSlug, step: stepNumber });
            return (
              <MenuItem
                key={stepNumber}
                stepContent={stepContent}
                isCompleted={isCompleted}
                isActive={isActive}
                url={url}
              />
            );
          })}
        </ul>
      </nav>
      <NewLink
        className="text-muted hover:decoration-subtle mx-1.5 my-1"
        size="small"
        iconBefore={<RiExternalLinkLine className="h-4 w-4" />}
        href="https://www.inngest.com/docs"
      >
        See documentation
      </NewLink>
      <NewLink
        className="text-muted hover:decoration-subtle mx-1.5 my-1"
        size="small"
        iconBefore={<RiDiscordLine className="h-4 w-4" />}
        href="https://www.inngest.com/discord"
      >
        Join discord community
      </NewLink>
      <NewLink
        className="text-muted hover:decoration-subtle mx-1.5 my-1"
        size="small"
        iconBefore={<RiMailLine className="h-4 w-4" />}
        href={pathCreator.support()}
      >
        Request a demo
      </NewLink>
    </div>
  );
}

const MenuItem = ({
  stepContent,
  isCompleted,
  isActive,
  url,
}: {
  stepContent: OnboardingMenuStepContentProps;
  isCompleted: boolean;
  isActive: boolean;
  url: Route;
}) => {
  const { title, description, icon: Icon } = stepContent;
  return (
    <Link href={url}>
      <li className="bg-canvasBase hover:bg-canvasSubtle group flex items-center gap-4 rounded-md p-1.5">
        <div
          className={cn(
            'group-hover:bg-contrast box-border flex h-[38px] w-[38px] items-center justify-center rounded-md border group-hover:border-none',
            isActive
              ? isCompleted
                ? 'border-primary-moderate bg-primary-3xSubtle group-hover:bg-primary-moderate'
                : 'border-contrast'
              : isCompleted
              ? 'bg-primary-3xSubtle group-hover:bg-primary-moderate border-none'
              : 'border-muted'
          )}
        >
          {isCompleted ? (
            <RiCheckboxCircleFill className="text-primary-moderate group-hover:text-alwaysWhite" />
          ) : (
            <Icon className="group-hover:text-onContrast h-5 w-5" />
          )}
        </div>
        <div>
          <h4 className="text-sm font-medium">{title}</h4>
          <p className="text-muted text-sm">{description}</p>
        </div>
      </li>
    </Link>
  );
};
