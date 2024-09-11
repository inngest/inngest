import Link from 'next/link';
import { NewButton } from '@inngest/components/Button';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import SegmentedProgressBar from '@inngest/components/ProgressBar/SegmentedProgressBar';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiBookReadLine, RiCheckboxCircleFill, RiCloseLine } from '@remixicon/react';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { EnvironmentType } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { onboardingWidgetContent } from '../Onboarding/content';
import useOnboardingStep from '../Onboarding/useOnboardingStep';

export default function OnboardingWidget({
  collapsed,
  closeWidget,
}: {
  collapsed: boolean;
  closeWidget: () => void;
}) {
  const { value: onboardingFlow } = useBooleanFlag('onboarding-flow-cloud');
  const { lastCompletedStep, isFinalStep, nextStep } = useOnboardingStep();

  const stepContent = onboardingWidgetContent.step[lastCompletedStep];

  if (!onboardingFlow) return;
  return (
    <>
      {collapsed && (
        <MenuItem
          href={pathCreator.onboardingSteps({
            envSlug: EnvironmentType.Production.toLowerCase(),
            step: nextStep,
          })}
          className="border-muted border"
          collapsed={collapsed}
          text="Onboarding guide"
          icon={<RiBookReadLine className="h-[18px] w-[18px]" />}
        />
      )}

      {!collapsed && (
        <Link
          href={pathCreator.onboardingSteps({
            envSlug: EnvironmentType.Production.toLowerCase(),
            step: nextStep,
          })}
          className="text-basis bg-canvasBase hover:bg-canvasSubtle border-subtle mb-5 block rounded border p-3 leading-tight"
        >
          <div className="flex h-[110px] flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <p
                  className={`${
                    isFinalStep && 'text-success'
                  } flex items-center gap-0.5 font-medium`}
                >
                  {isFinalStep && <RiCheckboxCircleFill className="text-success h-5 w-5" />}
                  {stepContent.title}
                </p>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <NewButton
                      icon={<RiCloseLine className="text-subtle" />}
                      kind="secondary"
                      appearance="ghost"
                      size="small"
                      className="hover:bg-canvasBase"
                      onClick={() => closeWidget()}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="right" className="dark max-w-40">
                    <p>{onboardingWidgetContent.tooltip.close}</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="text-subtle text-sm">{stepContent.description}</p>
            </div>
            {!isFinalStep && (
              <SegmentedProgressBar segmentsCompleted={lastCompletedStep} segments={4} />
            )}
            {stepContent.eta && (
              <p className="text-light text-[10px] font-medium uppercase">{stepContent.eta}</p>
            )}
            {stepContent.cta && (
              <NewButton
                appearance="outlined"
                className="hover:bg-canvasBase w-full text-sm"
                label={stepContent.cta}
                href="/settings/billing"
              />
            )}
          </div>
        </Link>
      )}
    </>
  );
}
