import { useMenuInputModality } from '@inngest/components/DropdownMenu/useMenuInputModality';
import {
  headlessMenuItemClassName,
  menuPanelClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import { Button } from '@inngest/components/Button';
import { RiArchive2Line, RiMore2Line } from '@remixicon/react';

export type EventActions = {
  archive: () => void;
};

export const ActionsMenu = ({ archive }: EventActions) => {
  const keyboardNavigation = useMenuInputModality();
  return (
    <Listbox>
      <Listbox.Button as="div">
        <Button
          kind="primary"
          appearance="outlined"
          size="medium"
          icon={<RiMore2Line />}
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options
          data-keyboard-navigation={keyboardNavigation}
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute right-1 top-5 z-50 w-[170px]',
          )}
        >
          <Listbox.Option
            className={cn(headlessMenuItemClassName)}
            value="eventsActions"
          >
            <Button
              appearance="ghost"
              kind="danger"
              size="medium"
              icon={<RiArchive2Line className="h-4 w-4" />}
              iconSide="left"
              label={'Archive event'}
              className="m-0 w-full justify-start"
              onClick={archive}
            />
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
