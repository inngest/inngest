import EmptyCard from '@inngest/components/Apps/EmptyCard';
import { Button } from '@inngest/components/Button/NewButton';
import { RiAddLine, RiExternalLinkLine } from '@remixicon/react';

import { pathCreator } from '@/utils/urls';
import useOnboardingStep from '../Onboarding/useOnboardingStep';
import { useNavigate } from '@tanstack/react-router';

export function EmptyOnboardingCard() {
  const navigate = useNavigate();
  const { nextStep, lastCompletedStep } = useOnboardingStep();

  return (
    <EmptyCard
      title="Sync your first Inngest App"
      description={
        <>
          In Inngest, an app is a group of functions served on a single endpoint
          or server. The first step is to create your app and functions, serve
          it, and test it locally with the Inngest Dev Server.
        </>
      }
      actions={
        <Button
          label="Get started"
          onClick={() =>
            navigate({
              to: pathCreator.onboardingSteps({
                step: nextStep ? nextStep.name : lastCompletedStep?.name,
                ref: 'app-apps-empty',
              }),
            })
          }
        />
      }
    />
  );
}

export function EmptyActiveCard({ envSlug }: { envSlug: string }) {
  const navigate = useNavigate();

  return (
    <EmptyCard
      title="No active apps found"
      description={
        <>
          Inngest lets you manage function deployments through apps. Sync your
          first app to display it here.
        </>
      }
      actions={
        <>
          <Button
            appearance="outlined"
            label="Go to docs"
            href="https://www.inngest.com/docs/apps/cloud"
            target="_blank"
            icon={<RiExternalLinkLine />}
            iconSide="left"
          />
          <Button
            label="Sync new app"
            icon={<RiAddLine />}
            iconSide="left"
            onClick={() => navigate({ to: pathCreator.createApp({ envSlug }) })}
          />
        </>
      }
    />
  );
}

export function EmptyArchivedCard() {
  return (
    <EmptyCard
      title="No archived apps found"
      description={
        <>
          Apps can be archived and unarchived at any time. Once an app is
          archived, all of its functions are archived.
        </>
      }
      actions={
        <>
          <Button
            label="Learn more"
            href="https://www.inngest.com/docs/apps/cloud"
            target="_blank"
            icon={<RiExternalLinkLine />}
            iconSide="left"
          />
        </>
      }
    />
  );
}
