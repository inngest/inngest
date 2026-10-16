import { useMenuInputModality } from '@inngest/components/DropdownMenu/useMenuInputModality';
import {
  menuItemClassName,
  menuPanelClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';

export const ProfileMenu = ({ children }: React.PropsWithChildren) => {
  const keyboardNavigation = useMenuInputModality();
  return (
    <Listbox>
      <Listbox.Button className="w-full cursor-pointer ring-0">
        {children as JSX.Element}
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options
          data-keyboard-navigation={keyboardNavigation}
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute -right-48 bottom-4 z-50 ml-8 w-[199px]',
          )}
        >
          <>
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
          </>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
