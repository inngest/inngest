import { Listbox } from '@headlessui/react';
import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';
import { RiLogoutBoxRLine } from '@remixicon/react';

import { useAuthStatusQuery, useLogoutMutation } from '@/store/authApi';

export const ProfileMenu = ({ children }: React.PropsWithChildren) => {
  const { data: authStatus } = useAuthStatusQuery();
  const [logout] = useLogoutMutation();

  const handleLogout = async () => {
    await logout().unwrap();
    window.location.href = '/login';
  };

  return (
    <Listbox>
      <Listbox.Button className="w-full cursor-pointer ring-0">
        {children as JSX.Element}
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute -right-48 bottom-4 z-50 ml-8 w-[199px] rounded border ring-0 focus:outline-none">
          <>
            <Listbox.Option
              disabled
              value="themeMode"
              className="text-muted mx-2 my-2 flex h-8 items-center justify-between px-2 text-[13px]"
            >
              <div>Theme</div>
              <ModeSwitch />
            </Listbox.Option>
            {authStatus?.authRequired && (
              <Listbox.Option
                value="logout"
                className="text-muted mx-2 mb-2 flex h-8 cursor-pointer items-center gap-2 rounded px-2 text-[13px] hover:bg-canvasSubtle"
                onClick={handleLogout}
              >
                <RiLogoutBoxRLine className="h-4 w-4" />
                <div>Sign out</div>
              </Listbox.Option>
            )}
          </>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
