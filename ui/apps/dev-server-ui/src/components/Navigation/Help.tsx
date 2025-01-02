'use client';

import NextLink from 'next/link';
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
        <Listbox.Options className="bg-canvasBase border-subtle absolute -right-48 bottom-0 z-50 ml-8 w-[199px] gap-y-0.5 rounded border shadow ring-0 focus:outline-none">
          <NextLink href="https://www.inngest.com/docs/local-development" target="_blank">
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="docs"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiExternalLinkLine className="text-muted mr-2 h-4 w-4 " />
                <div>Inngest Documentation</div>
              </div>
            </Listbox.Option>
          </NextLink>
          <NextLink href="https://app.inngest.com/support" target="_blank">
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="support"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiMailLine className="text-muted mr-2 h-4 w-4" />
                <div>Support</div>
              </div>
            </Listbox.Option>
          </NextLink>
          <NextLink href="https://www.inngest.com/discord" target="_blank">
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 my-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="discord"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiDiscordLine className="text-muted mr-2 h-4 w-4" />
                <div>Join Discord</div>
              </div>
            </Listbox.Option>
          </NextLink>
          <NextLink href="https://roadmap.inngest.com/roadmap" target="_blank">
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 my-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="roadmap"
            >
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiRoadMapLine className="text-muted mr-2 h-4 w-4" />
                <div>Inngest Roadmap</div>
              </div>
            </Listbox.Option>
          </NextLink>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
