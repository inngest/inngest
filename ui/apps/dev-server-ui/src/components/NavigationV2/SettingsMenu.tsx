import { useMenuInputModality } from '@inngest/components/DropdownMenu/useMenuInputModality';
import {
  menuItemClassName,
  menuPanelClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';
import { RiSettings3Line } from '@remixicon/react';

export const SettingsMenu = () => {
  const keyboardNavigation = useMenuInputModality();
  return (
    <Listbox>
      <div className="relative flex">
        <Listbox.Button
          aria-label="Settings"
          className="text-muted hover:bg-canvasBase hover:text-basis flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center rounded ring-0"
        >
          <RiSettings3Line className="h-[18px] w-[18px]" />
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
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
