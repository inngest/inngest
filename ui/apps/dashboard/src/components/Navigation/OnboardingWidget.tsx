import Link from 'next/link';
import { NewButton } from '@inngest/components/Button';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import SegmentedProgressBar from '@inngest/components/ProgressBar/SegmentedProgressBar';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiBookReadLine, RiCheckboxCircleFill, RiCloseLine } from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import { pathCreator } from '@/utils/urls';
import { onboardingWidgetContent } from '../Onboarding/content';
import { type OnboardingSteps } from '../Onboarding/types';

type OnboardingStepsString = `${OnboardingSteps}`;

export default function OnboardingWidget({ collapsed }: { collapsed: boolean }) {
  const { value: onBoardingFlow } = useBooleanFlag('onboarding-flow-cloud');

  const [isOnboardingWidgetOpen, setIsOnboardingWidgetOpen] = useLocalStorage(
    'showOnboardingWidget',
    true
  );

  const [onboardingWidgetStep] = useLocalStorage<OnboardingStepsString>(
    'onboardingWidgetStep',
    '1'
  );
  const currentStep: OnboardingSteps = Number(onboardingWidgetStep) as OnboardingSteps;
  const finalStep = currentStep === 4;
  const stepContent = onboardingWidgetStep
    ? onboardingWidgetContent.step[currentStep]
    : onboardingWidgetContent.step[1];

  if (!onBoardingFlow) return;
  return (
    isOnboardingWidgetOpen && (
      <>
        {collapsed && (
          <MenuItem
            href={pathCreator.onboarding()}
            className="border-muted border"
            collapsed={collapsed}
            text="Onboarding guide"
            icon={<RiBookReadLine className="h-[18px] w-[18px]" />}
          />
        )}

        {!collapsed && (
          <Link
            href={pathCreator.onboarding()}
            className="text-basis bg-canvasBase hover:bg-canvasSubtle border-subtle mb-5 block rounded border p-3 leading-tight"
          >
            <div className="flex h-[110px] flex-col justify-between">
              <div>
                <div className="flex items-center justify-between">
                  <p
                    className={`${
                      finalStep && 'text-success'
                    } flex items-center gap-px font-medium`}
                  >
                    {finalStep && <RiCheckboxCircleFill className="text-success h-5 w-5" />}
                    {stepContent.title}
                  </p>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <NewButton
                        icon={<RiCloseLine className="text-muted" />}
                        kind="secondary"
                        appearance="ghost"
                        size="small"
                        className="hover:bg-canvasBase"
                        onClick={() => setIsOnboardingWidgetOpen(false)}
                      />
                    </TooltipTrigger>
                    <TooltipContent side="right" className="dark max-w-40">
                      <p>{onboardingWidgetContent.tooltip.close}</p>
                    </TooltipContent>
                  </Tooltip>
                </div>
                <p className="text-muted text-sm">{stepContent.description}</p>
              </div>
              {!finalStep && <SegmentedProgressBar segmentsCompleted={currentStep} segments={4} />}
              {stepContent.eta && (
                <p className="text-light text-[10px] font-medium uppercase">{stepContent.eta}</p>
              )}
              {stepContent.cta && (
                <NewButton
                  appearance="outlined"
                  className="hover:bg-canvasBase w-full text-sm"
                  label={stepContent.cta}
                />
              )}
            </div>
          </Link>
        )}
      </>
    )
  );
}
