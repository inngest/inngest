import {
  RiBankCard2Line,
  RiCheckboxCircleFill,
  RiSmartphoneLine,
} from '@remixicon/react';
import { cn } from '@inngest/components/utils/classNames';
import type { VerificationState } from './types';

export function VerificationSteps({
  phone,
  card,
}: Pick<VerificationState, 'phone' | 'card'>) {
  const steps = [
    { label: 'Phone', complete: phone === 'complete', Icon: RiSmartphoneLine },
    { label: 'Card', complete: card === 'complete', Icon: RiBankCard2Line },
  ];
  const currentStep = steps.findIndex((step) => !step.complete);

  return (
    <ol
      aria-label="Verification progress"
      className="flex items-center gap-2 text-sm leading-5"
    >
      {steps.map(({ label, complete, Icon }, index) => (
        <li
          key={label}
          aria-current={index === currentStep ? 'step' : undefined}
          className={cn(
            'flex items-center gap-1',
            index === currentStep ? 'text-basis' : 'text-disabled',
          )}
        >
          {complete ? (
            <RiCheckboxCircleFill
              aria-hidden="true"
              className="text-primary-moderate h-3.5 w-3.5 shrink-0"
            />
          ) : (
            <Icon aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
          )}
          <span>{label}</span>
          {complete && <span className="sr-only">completed</span>}
          {!complete && index !== currentStep && (
            <span className="sr-only">upcoming</span>
          )}
          {index === 0 && (
            <span aria-hidden="true" className="bg-canvasMuted ml-1 h-px w-3" />
          )}
        </li>
      ))}
    </ol>
  );
}
