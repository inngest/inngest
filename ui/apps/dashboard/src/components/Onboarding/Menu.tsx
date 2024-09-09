'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { NewLink } from '@inngest/components/Link';
import { cn } from '@inngest/components/utils/classNames';
import { RiDiscordLine, RiExternalLinkLine, RiMailLine } from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { pathCreator } from '@/utils/urls';
import { type OnboardingSteps, type OnboardingStepsString } from '../Onboarding/types';
import { onboardingMenuStepContent } from './content';
import { type OnboardingMenuStepContent as OnboardingMenuStepContentProps } from './types';

export default function Menu({ envSlug }: { envSlug: string }) {
  const [onboardingLastStepCompleted] = useLocalStorage<OnboardingStepsString>(
    'onboardingLastStepCompleted',
    '1',
    { raw: true }
  );
  const lastCompletedStep: OnboardingSteps = Number(onboardingLastStepCompleted) as OnboardingSteps;
  const pathname = usePathname();

  const activeStep = pathname.split('/').pop() || '1';
  const stepNumbers: OnboardingSteps[] = [1, 2, 3, 4];

  return (
    <div className="flex flex-col">
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
            'group-hover:bg-contrast rounded-md border p-2',
            isActive
              ? isCompleted
                ? 'border-primary-moderate bg-primary-3xSubtle group-hover:bg-primary-moderate'
                : 'border-contrast'
              : 'border-muted'
          )}
        >
          <Icon className="group-hover:text-onContrast h-5 w-5" />
        </div>
        <div>
          <h4 className="text-sm font-medium">{title}</h4>
          <p className="text-muted text-sm">{description}</p>
        </div>
      </li>
    </Link>
  );
};
