import {
  headlessMenuItemClassName,
  menuPanelClassName,
} from '@inngest/components/DropdownMenu/menuStyles';
import { cn } from '@inngest/components/utils/classNames';
import { Listbox } from '@headlessui/react';
import { MenuItem } from '@inngest/components/Menu/MenuItem';
import {
  RiDiscordLine,
  RiExternalLinkLine,
  RiMailLine,
  RiQuestionLine,
  RiRoadMapLine,
} from '@remixicon/react';

export const Help = ({ collapsed }: { collapsed: boolean }) => {
  return (
    <Listbox>
      <Listbox.Button className="w-full ring-0">
        <MenuItem
          collapsed={collapsed}
          text="Help and Feedback"
          icon={<RiQuestionLine className="h-[18px] w-[18px]" />}
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options
          className={cn(
            menuPanelClassName,
            'dropdown-focus absolute -right-48 bottom-0 z-50 ml-8 w-[199px]',
          )}
        >
          <a
            href="https://www.inngest.com/docs/local-development"
            target="_blank"
          >
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="docs"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiExternalLinkLine className="text-muted mr-2 h-4 w-4 " />
                <div>Inngest Documentation</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://app.inngest.com/support" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="support"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiMailLine className="text-muted mr-2 h-4 w-4" />
                <div>Support</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://www.inngest.com/discord" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="discord"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiDiscordLine className="text-muted mr-2 h-4 w-4" />
                <div>Join Discord</div>
              </div>
            </Listbox.Option>
          </a>
          <a href="https://roadmap.inngest.com/roadmap" target="_blank">
            <Listbox.Option
              className={cn(headlessMenuItemClassName)}
              value="roadmap"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiRoadMapLine className="text-muted mr-2 h-4 w-4" />
                <div>Inngest Roadmap</div>
              </div>
            </Listbox.Option>
          </a>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
