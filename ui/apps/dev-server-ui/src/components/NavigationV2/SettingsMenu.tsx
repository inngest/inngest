import { Listbox } from '@headlessui/react';
import ModeSwitch from '@inngest/components/ThemeMode/ModeSwitch';
import { RiSettings3Line } from '@remixicon/react';

export const SettingsMenu = () => {
  return (
    <Listbox>
      <div className="relative flex">
        <Listbox.Button
          aria-label="Settings"
          className="text-muted hover:bg-canvasBase hover:text-basis flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center rounded ring-0"
        >
          <RiSettings3Line className="h-[18px] w-[18px]" />
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
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
