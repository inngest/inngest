import { useMenuInputModality } from '@inngest/components/DropdownMenu/useMenuInputModality';
import {
  headlessMenuItemClassName,
  menuItemClassName,
  menuPanelClassName,
  menuSeparatorClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import { useNavigate } from '@tanstack/react-router';
import { RiUserLine, RiUserSharedLine } from '@remixicon/react';

import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';

import type { FileRouteTypes } from '@/routeTree.gen';
import { SignOutButton } from '../Auth/SignOutButton';

type Props = React.PropsWithChildren<{
  isMarketplace: boolean;
}>;

const itemClassName = headlessMenuItemClassName;

export const ProfileMenu = ({ children, isMarketplace }: Props) => {
  const keyboardNavigation = useMenuInputModality();
  const navigate = useNavigate();

  return (
    <Listbox>
      <div className="relative flex">
        <Listbox.Button className="bg-canvasMuted/75 border border-subtle text-subtle flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-full text-xs uppercase ring-0">
          {children}
        </Listbox.Button>
        <Listbox.Options
          data-keyboard-navigation={keyboardNavigation}
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute right-0 top-full z-50 mt-2 w-[220px]',
          )}
        >
          <Listbox.Option
            disabled
            value="themeMode"
            className={cn(
              menuItemClassName,
              'text-muted min-h-10 justify-between',
            )}
          >
            <div>Theme</div>
            <ModeSwitch />
          </Listbox.Option>

          <hr className={menuSeparatorClassName} />

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

          <hr className={menuSeparatorClassName} />

          <Listbox.Option
            className={cn(headlessMenuItemClassName)}
            value="signOut"
          >
            <SignOutButton isMarketplace={isMarketplace} />
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
