import { Listbox } from '@headlessui/react';
import { useNavigate } from '@tanstack/react-router';
import { RiUserLine, RiUserSharedLine } from '@remixicon/react';

import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';

import type { FileRouteTypes } from '@/routeTree.gen';
import { SignOutButton } from '../Auth/SignOutButton';

type Props = React.PropsWithChildren<{
  isMarketplace: boolean;
}>;

const itemClassName =
  'text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]';

export const ProfileMenu = ({ children, isMarketplace }: Props) => {
  const navigate = useNavigate();

  return (
    <Listbox>
      <div className="relative">
        <Listbox.Button className="cursor-pointer ring-0">
          {children}
        </Listbox.Button>
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute right-0 top-full z-50 mt-2 w-[220px] rounded border ring-0 focus:outline-none">
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
            className={itemClassName}
            value="userProfile"
            onClick={() =>
              navigate({ to: '/settings/user/profile' as FileRouteTypes['to'] })
            }
          >
            <RiUserLine className="text-muted mr-2 h-4 w-4" />
            <div>Your Profile</div>
          </Listbox.Option>

          <Listbox.Option
            className={`${itemClassName} mb-2`}
            value="switchAccount"
            onClick={() =>
              navigate({ to: '/sign-in/choose' as FileRouteTypes['to'] })
            }
          >
            <RiUserSharedLine className="text-muted mr-2 h-4 w-4" />
            <div>Switch account</div>
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
