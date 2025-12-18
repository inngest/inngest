import { Listbox } from '@headlessui/react';
import {
  RiArrowLeftRightLine,
  RiBillLine,
  RiEqualizerLine,
  RiGroupLine,
  RiUserLine,
  RiUserSharedLine,
} from '@remixicon/react';

import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';

import { pathCreator } from '@/utils/urls';
import { useNavigate } from '@tanstack/react-router';
import { SignOutButton } from '../Auth/SignOutButton';
import type { FileRouteTypes } from '@/routeTree.gen';

type Props = React.PropsWithChildren<{
  isMarketplace: boolean;
}>;

export const ProfileMenu = ({ children, isMarketplace }: Props) => {
  const navigate = useNavigate();

  return (
    <Listbox>
      <Listbox.Button className="w-full cursor-pointer ring-0">
        {children}
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute -right-48 bottom-4 z-50 ml-8 w-[199px] rounded border ring-0 focus:outline-none">
          <Listbox.Option
            disabled
            value="themeMode"
            className="text-muted mx-2 my-2 flex h-8 items-center justify-between px-2 text-[13px]"
          >
            <div>Theme</div>
            <ModeSwitch />
          </Listbox.Option>

          <hr className="border-subtle" />

          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="userProfile"
            onClick={() => navigate({ to: '/settings/user' })}
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiUserLine className="text-muted mr-2 h-4 w-4" />
              <div>Your Profile</div>
            </div>
          </Listbox.Option>
          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="org"
            onClick={() =>
              navigate({ to: '/settings/organization' as FileRouteTypes['to'] })
            }
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiEqualizerLine className="text-muted mr-2 h-4 w-4 " />
              <div>Your Organization</div>
            </div>
          </Listbox.Option>

          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="members"
            onClick={() =>
              navigate({
                to: '/settings/organization/organization-members' as FileRouteTypes['to'],
              })
            }
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiGroupLine className="text-muted mr-2 h-4 w-4" />
              <div>Members</div>
            </div>
          </Listbox.Option>

          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="billing"
            onClick={() => navigate({ to: pathCreator.billing() })}
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiBillLine className="text-muted mr-2 h-4 w-4" />
              <div>Billing</div>
            </div>
          </Listbox.Option>

          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="switchOrg"
            onClick={() =>
              navigate({ to: '/organization-list' as FileRouteTypes['to'] })
            }
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiArrowLeftRightLine className="text-muted mr-2 h-4 w-4" />
              <div>Switch Organization</div>
            </div>
          </Listbox.Option>

          <hr className="border-subtle" />

          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle m-2 mx-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="switchAccount"
            onClick={() =>
              navigate({ to: '/sign-in/choose' as FileRouteTypes['to'] })
            }
          >
            <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
              <RiUserSharedLine className="text-muted mr-2 h-4 w-4" />
              <div>Switch Account</div>
            </div>
          </Listbox.Option>
          <hr className="border-subtle" />
          <Listbox.Option
            className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
            value="signOut"
          >
            <SignOutButton isMarketplace={isMarketplace} />
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
