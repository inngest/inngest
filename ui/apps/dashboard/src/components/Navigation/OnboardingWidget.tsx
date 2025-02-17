import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import SegmentedProgressBar from '@inngest/components/ProgressBar/SegmentedProgressBar';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiBookReadLine, RiCheckboxCircleFill, RiCloseLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import { onboardingWidgetContent } from '../Onboarding/content';
import { OnboardingSteps, steps } from '../Onboarding/types';
import useOnboardingStep from '../Onboarding/useOnboardingStep';
import { useOnboardingTracking } from '../Onboarding/useOnboardingTracking';

export default function OnboardingWidget({
  collapsed,
  closeWidget,
}: {
  collapsed: boolean;
  closeWidget: () => void;
}) {
  const router = useRouter();
  const { lastCompletedStep, nextStep, totalStepsCompleted } = useOnboardingStep();
  const tracking = useOnboardingTracking();

  const stepContent = lastCompletedStep?.isFinalStep
    ? onboardingWidgetContent.step.success
    : onboardingWidgetContent.step[nextStep?.name || OnboardingSteps.CreateApp];

  return (
    <>
      {collapsed && (
        <MenuItem
          href={pathCreator.onboardingSteps({
            step: nextStep ? nextStep.name : lastCompletedStep?.name,
            ref: 'app-onboarding-widget',
          })}
          className="border-muted border"
          collapsed={collapsed}
          text="Onboarding guide"
          icon={<RiBookReadLine className="h-[18px] w-[18px]" />}
        />
      )}

      {!collapsed && (
        <NextLink
          href={pathCreator.onboardingSteps({
            step: nextStep ? nextStep.name : lastCompletedStep?.name,
            ref: 'app-onboarding-widget',
          })}
          className="text-basis bg-canvasBase hover:bg-canvasSubtle border-subtle mb-5 block rounded border p-3 leading-tight"
        >
          <div className="flex h-[110px] flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <p
                  className={`${
                    lastCompletedStep?.isFinalStep && 'text-success'
                  } flex items-center gap-0.5 font-medium`}
                >
                  {lastCompletedStep?.isFinalStep && (
                    <RiCheckboxCircleFill className="text-success h-5 w-5" />
                  )}
                  {stepContent.title}
                </p>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      icon={<RiCloseLine className="text-subtle" />}
                      kind="secondary"
                      appearance="ghost"
                      size="small"
                      className="hover:bg-canvasBase"
                      onClick={(e) => {
                        e.preventDefault();
                        tracking?.trackOnboardingAction(undefined, {
                          metadata: {
                            type: 'btn-click',
                            label: 'close-widget',
                            totalStepsCompleted: totalStepsCompleted,
                          },
                        });
                        closeWidget();
                      }}
                    />
                  </TooltipTrigger>
                  <TooltipContent side="right" className="dark max-w-40">
                    <p>{onboardingWidgetContent.tooltip.close}</p>
                  </TooltipContent>
                </Tooltip>
              </div>
              <p className="text-subtle text-sm">{stepContent.description}</p>
            </div>
            {!lastCompletedStep?.isFinalStep && (
              <SegmentedProgressBar
                segmentsCompleted={totalStepsCompleted}
                segments={steps.length}
              />
            )}
            {stepContent.eta && (
              <p className="text-light text-[10px] font-medium uppercase">{stepContent.eta}</p>
            )}
            {stepContent.cta && (
              <Button
                appearance="outlined"
                className="hover:bg-canvasBase w-full text-sm"
                label={stepContent.cta}
                onClick={(e) => {
                  e.preventDefault();
                  router.push(pathCreator.billing() + '?ref=app-onboarding-widget');
                }}
              />
            )}
          </div>
        </NextLink>
      )}
    </>
  );
}
