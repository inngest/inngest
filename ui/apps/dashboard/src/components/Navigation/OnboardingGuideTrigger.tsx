import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { RiBookReadLine } from '@remixicon/react';
import { Link } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';
import useOnboardingStep from '../Onboarding/useOnboardingStep';

export default function OnboardingGuideTrigger({
  collapsed,
  showWidget,
}: {
  collapsed: boolean;
  showWidget: () => void;
}) {
  const { nextStep, lastCompletedStep } = useOnboardingStep();
  const to = pathCreator.onboardingSteps({
    step: nextStep ? nextStep.name : lastCompletedStep?.name,
    ref: 'app-sidebar-onboarding',
  });

  return (
    <Link to={to} onClick={() => showWidget()}>
      <OptionalTooltip tooltip={collapsed ? 'Onboarding guide' : ''}>
        <div className="hover:bg-canvasSubtle text-subtle hover:text-basis my-0.5 flex h-8 w-full flex-row items-center rounded px-1.5">
          <RiBookReadLine className="h-[18px] w-[18px]" />
          {!collapsed && (
            <span className="ml-2.5 text-sm leading-tight">
              Onboarding guide
            </span>
          )}
        </div>
      </OptionalTooltip>
    </Link>
  );
}
