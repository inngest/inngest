import { Listbox } from '@headlessui/react';
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
import type { ProfileDisplayType } from '@/queries/server/profile';
import { pathCreator } from '@/utils/urls';
import useOnboardingStep from '../Onboarding/useOnboardingStep';

type Props = React.PropsWithChildren<{
  profile: ProfileDisplayType;
  showOnboardingWidget: () => void;
}>;

const itemClassName =
  'text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]';

export const OrgMenu = ({ children, profile, showOnboardingWidget }: Props) => {
  const navigate = useNavigate();
  const { nextStep, lastCompletedStep } = useOnboardingStep();
  const orgName = profile.orgName ?? '';

  const onboardingTo = pathCreator.onboardingSteps({
    step: nextStep ? nextStep.name : lastCompletedStep?.name,
    ref: 'app-org-menu-onboarding',
  });

  return (
    <Listbox>
      <div className="relative flex">
        <Listbox.Button className="text-basis hover:bg-canvasMuted flex h-7 cursor-pointer items-center gap-1.5 rounded px-2 text-sm ring-0">
          {children}
        </Listbox.Button>
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute left-0 top-full z-50 mt-2 w-[240px] rounded border ring-0 focus:outline-none">
          <div
            className="text-basis px-3 pt-3 pb-2 text-sm font-medium"
            title={orgName}
          >
            {orgName}
          </div>
          <hr className="border-subtle" />

          <Listbox.Option
            className={itemClassName}
            value="settings"
            onClick={() =>
              navigate({ to: '/settings/organization' as FileRouteTypes['to'] })
            }
          >
            <RiEqualizerLine className="text-muted mr-2 h-4 w-4" />
            <div>Settings</div>
          </Listbox.Option>

          <Listbox.Option
            className={itemClassName}
            value="members"
            onClick={() =>
              navigate({
                to: '/settings/organization/organization-members' as FileRouteTypes['to'],
              })
            }
          >
            <RiGroupLine className="text-muted mr-2 h-4 w-4" />
            <div>Members</div>
          </Listbox.Option>

          <Listbox.Option
            className={itemClassName}
            value="billing"
            onClick={() => navigate({ to: pathCreator.billing() })}
          >
            <RiBillLine className="text-muted mr-2 h-4 w-4" />
            <div>Billing</div>
          </Listbox.Option>

          <Listbox.Option
            className={itemClassName}
            value="integrations"
            onClick={() =>
              navigate({ to: '/settings/integrations' as FileRouteTypes['to'] })
            }
          >
            <RiPlugLine className="text-muted mr-2 h-4 w-4" />
            <div>Integrations</div>
          </Listbox.Option>

          <Listbox.Option
            className={itemClassName}
            value="apiKeys"
            onClick={() =>
              navigate({ to: '/settings/api-keys' as FileRouteTypes['to'] })
            }
          >
            <RiKey2Line className="text-muted mr-2 h-4 w-4" />
            <div>API keys</div>
          </Listbox.Option>

          <Listbox.Option
            className={itemClassName}
            value="onboardingGuide"
            onClick={() => {
              showOnboardingWidget();
              navigate({ to: onboardingTo });
            }}
          >
            <RiBookReadLine className="text-muted mr-2 h-4 w-4" />
            <div>Onboarding guide</div>
          </Listbox.Option>

          <hr className="border-subtle mt-2" />

          <Listbox.Option
            className={`${itemClassName} mb-2`}
            value="switchOrg"
            onClick={() =>
              navigate({ to: '/organization-list' as FileRouteTypes['to'] })
            }
          >
            <RiArrowLeftRightLine className="text-muted mr-2 h-4 w-4" />
            <div>Switch organisations</div>
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
