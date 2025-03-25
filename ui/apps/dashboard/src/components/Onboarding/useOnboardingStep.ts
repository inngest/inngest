import { useEffect, useMemo, useState } from 'react';

import {
  steps,
  type OnboardingStep,
  type OnboardingSteps,
  type TotalStepsCompleted,
} from './types';
import { useOnboardingStepCompletedTracking } from './useOnboardingTracking';

const getHighestStep = (steps: OnboardingStep[]): OnboardingStep | null => {
  return steps.length > 0
    ? steps.reduce((prev, current) => (current.stepNumber > prev.stepNumber ? current : prev))
    : null;
};

export default function useOnboardingStep() {
  // Temporary approach, we will store this value in the backend in the future
  const [lastCompletedStep, setLastCompletedStep] = useState<OnboardingStep | null>(null);
  const [completedSteps, setCompletedSteps] = useState<OnboardingStep[] | []>([]);

  const tracking = useOnboardingStepCompletedTracking();

  useEffect(() => {
    const storedSteps = localStorage.getItem('onboardingCompletedSteps');
    if (storedSteps) {
      const parsedSteps: OnboardingStep[] = JSON.parse(storedSteps);
      setCompletedSteps(parsedSteps);

      // Set the initial lastCompletedStep based on the highest stepNumber in parsedSteps
      const highestStep = getHighestStep(parsedSteps);
      setLastCompletedStep(highestStep);
    }

    const handleStorageChange = (event: StorageEvent) => {
      if (event.key === 'onboardingCompletedSteps') {
        const newValue = event.newValue ? JSON.parse(event.newValue) : [];
        setLastCompletedStep(newValue);

        // Update lastCompletedStep
        const highestStep = getHighestStep(newValue);
        setLastCompletedStep(highestStep);
      }
    };

    // Listen for storage events from other components
    window.addEventListener('storage', handleStorageChange);

    // Custom event for same-window updates
    const handleCustomEvent = (event: CustomEvent) => {
      const newCompletedSteps: OnboardingStep[] = event.detail;
      setCompletedSteps(newCompletedSteps);

      // Update lastCompletedStep
      const highestStep = getHighestStep(newCompletedSteps);
      setLastCompletedStep(highestStep);
    };

    window.addEventListener('onboardingStepUpdate', handleCustomEvent as EventListener);

    return () => {
      window.removeEventListener('storage', handleStorageChange);
      window.removeEventListener('onboardingStepUpdate', handleCustomEvent as EventListener);
    };
  }, []);

  const nextStep = useMemo(() => {
    if (!lastCompletedStep) {
      // If no step has been completed, return the first step
      return steps.find((step) => step.stepNumber === 1) || null;
    }
    return steps.find((step) => step.stepNumber === lastCompletedStep.stepNumber + 1) || null;
  }, [lastCompletedStep]);

  const totalStepsCompleted: TotalStepsCompleted = completedSteps.length;

  const updateCompletedSteps = (stepName: OnboardingSteps, metadata?: Record<string, any>) => {
    if (typeof window !== 'undefined') {
      const step = steps.find((s) => s.name === stepName);

      if (!step) {
        console.warn(`Step with name ${stepName} not found.`);
        return;
      }

      // Avoid adding duplicate steps by name
      if (!completedSteps.some((s) => s.name === step.name)) {
        // If we have previous steps not completed yet by the user, we automatically mark them as completed
        const stepsToAdd = steps.filter(
          (s) =>
            s.stepNumber < step.stepNumber &&
            !completedSteps.some((cs) => cs.stepNumber === s.stepNumber)
        );

        const newCompletedSteps = [...completedSteps, ...stepsToAdd, step];

        // Update local state
        setCompletedSteps(newCompletedSteps);
        setLastCompletedStep(step);

        // Update localStorage
        localStorage.setItem('onboardingCompletedSteps', JSON.stringify(newCompletedSteps));

        // Dispatch event for other components
        window.dispatchEvent(
          new CustomEvent('onboardingStepUpdate', { detail: newCompletedSteps })
        );

        // Tracking for the previous steps marked with automatic completion
        stepsToAdd.forEach((s) => {
          tracking?.trackOnboardingStepCompleted(s, {
            metadata: { completionSource: 'automatic' },
          });
        });

        tracking?.trackOnboardingStepCompleted(step, metadata);
      }
    }
  };

  return { lastCompletedStep, completedSteps, updateCompletedSteps, nextStep, totalStepsCompleted };
}
