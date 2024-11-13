'use client';

import { useRouter } from 'next/navigation';
import { NewButton } from '@inngest/components/Button';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';

import { pathCreator } from '@/utils/urls';
import useOnboardingStep from './useOnboardingStep';

export default function EmptyAppsCard() {
  const router = useRouter();
  const { nextStep, lastCompletedStep } = useOnboardingStep();

  return (
    <div className="border-muted bg-canvasSubtle flex flex-col items-center gap-6 rounded-lg border border-dashed px-6 py-9">
      <div className="bg-primary-3xSubtle text-primary-moderate rounded-lg p-3 ">
        <AppsIcon className="h-8 w-8" />
      </div>
      <div className="text-center">
        <p className="mb-2 text-2xl">Sync your first Inngest App</p>
        <p className="max-w-3xl">
          In Inngest, an app is a group of functions served on a single endpoint or server. The
          first step is to create your app and functions, serve it, and test it locally with the
          Inngest Dev Server.
        </p>
      </div>
      <NewButton
        label="Take me to onboarding"
        onClick={() =>
          router.push(
            pathCreator.onboardingSteps({
              step: nextStep ? nextStep.name : lastCompletedStep?.name,
              ref: 'app-apps-empty',
            })
          )
        }
      />
    </div>
  );
}
