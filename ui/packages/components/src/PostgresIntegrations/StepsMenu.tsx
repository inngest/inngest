'use client';

import StepsMenu from '@inngest/components/Steps/StepsMenu';

import { isValidStep, type PostgresIntegrationMenuContent } from './types';

export default function Menu({
  stepsCompleted,
  activeStep,
  content,
  links,
  steps,
  pathname,
}: {
  stepsCompleted: string[];
  activeStep: string;
  content: PostgresIntegrationMenuContent;
  links: React.ReactNode;
  steps: string[];
  pathname: string;
}) {
  const { step, title } = content;
  return (
    <StepsMenu title={title} links={links}>
      {steps.map((stepKey) => {
        if (!isValidStep(stepKey)) {
          return 'error';
        }
        const stepContent = step[stepKey];
        const isCompleted = stepsCompleted.includes(stepKey);
        const isActive = activeStep === stepKey;
        const url = `${pathname}/${stepKey}`;
        return (
          <StepsMenu.MenuItem
            isDisabled={!isCompleted && !isActive}
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
