import { useState } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { useNavigate } from '@tanstack/react-router';
import {
  RiArrowLeftRightLine,
  RiBillLine,
  RiBookReadLine,
  RiEqualizerLine,
  RiGroupLine,
  RiKey2Line,
  RiPlugLine,
} from '@remixicon/react';

import type { FileRouteTypes } from '@/routeTree.gen';
import { GetCurrentPlanDocument } from '@/gql/graphql';
import type { ProfileDisplayType } from '@/queries/server/profile';
import { useSkippableGraphQLQuery } from '@/utils/useGraphQLQuery';
import { pathCreator } from '@/utils/urls';
import useOnboardingStep from '../Onboarding/useOnboardingStep';
import OrgAvatar from './OrgAvatar';

type Props = React.PropsWithChildren<{
  profile: ProfileDisplayType;
  showOnboardingWidget: () => void;
}>;

const iconClassName = 'text-muted h-4 w-4';

export const OrgMenu = ({ children, profile, showOnboardingWidget }: Props) => {
  const navigate = useNavigate();
  const { nextStep, lastCompletedStep } = useOnboardingStep();
  const orgName = profile.orgName ?? '';

  // The top bar renders on every page, so hold the plan query until the menu is
  // actually opened.
  const [open, setOpen] = useState(false);
  const { data, isLoading } = useSkippableGraphQLQuery({
    query: GetCurrentPlanDocument,
    variables: {},
    skip: !open,
  });
  const planName = data?.account.plan?.name;

  const onboardingTo = pathCreator.onboardingSteps({
    step: nextStep ? nextStep.name : lastCompletedStep?.name,
    ref: 'app-org-menu-onboarding',
  });

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger className="text-basis hover:bg-canvasMuted flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-sm leading-none ring-0">
        {children}
      </DropdownMenuTrigger>

      <DropdownMenuContent className="w-[212px]">
        <DropdownMenuLabel className="flex items-center gap-2.5 p-2">
          <OrgAvatar profile={profile} size="lg" />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span
              className="text-basis truncate text-sm font-medium"
              title={orgName}
            >
              {orgName}
            </span>
            {isLoading ? (
              <Skeleton className="h-3 w-16" />
            ) : (
              planName && (
                <span className="text-muted truncate text-xs" title={planName}>
                  {planName}
                </span>
              )
            )}
          </div>
        </DropdownMenuLabel>

        <DropdownMenuItem
          onSelect={() =>
            navigate({ to: '/settings/organization' as FileRouteTypes['to'] })
          }
        >
          <RiEqualizerLine className={iconClassName} />
          Settings
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={() =>
            navigate({
              to: '/settings/organization/organization-members' as FileRouteTypes['to'],
            })
          }
        >
          <RiGroupLine className={iconClassName} />
          Members
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={() => navigate({ to: pathCreator.billing() })}
        >
          <RiBillLine className={iconClassName} />
          Billing
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={() =>
            navigate({ to: '/settings/integrations' as FileRouteTypes['to'] })
          }
        >
          <RiPlugLine className={iconClassName} />
          Integrations
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={() =>
            navigate({ to: '/settings/api-keys' as FileRouteTypes['to'] })
          }
        >
          <RiKey2Line className={iconClassName} />
          API keys
        </DropdownMenuItem>

        <DropdownMenuItem
          onSelect={() => {
            showOnboardingWidget();
            navigate({ to: onboardingTo });
          }}
        >
          <RiBookReadLine className={iconClassName} />
          Onboarding guide
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem
          onSelect={() =>
            navigate({ to: '/organization-list' as FileRouteTypes['to'] })
          }
        >
          <RiArrowLeftRightLine className={iconClassName} />
          Switch organisations
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
