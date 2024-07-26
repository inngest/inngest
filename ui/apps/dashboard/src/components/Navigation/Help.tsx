'use client';

import Link from 'next/link';
import { Listbox } from '@headlessui/react';
import {
  RiArticleLine,
  RiDiscordLine,
  RiExternalLinkLine,
  RiMailLine,
  RiQuestionLine,
  RiRoadMapLine,
} from '@remixicon/react';

import { useSystemStatus } from '@/app/(organization-active)/support/statusPage';
import { MenuItem } from './MenuItem';
import SystemStatusIcon from './old/SystemStatusIcon';

export const Help = ({ collapsed }: { collapsed: boolean }) => {
  const status = useSystemStatus();

  return (
    <div className="m-2.5">
      <Listbox>
        <Listbox.Button className="w-full ring-0">
          <MenuItem
            collapsed={collapsed}
            text="Help and Feedback"
            icon={<RiQuestionLine className="text-muted h-[18px] w-[18px]" />}
          />
        </Listbox.Button>
        <div className="relative">
          <Listbox.Options className="bg-canvasBase absolute -right-48 bottom-0 z-50 ml-8 w-[199px] rounded border shadow ring-0 focus:outline-none">
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="https://www.inngest.com/docs?ref=support-center">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiExternalLinkLine className="text-subtle mr-2 h-4 w-4 " />
                  <div>Inngest Documentation</div>
                </div>
              </Link>
            </Listbox.Option>
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="/support">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiMailLine className="text-subtle mr-2 h-4 w-4" />
                  <div>Support</div>
                </div>
              </Link>
            </Listbox.Option>
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle border-subtle flex h-12 cursor-pointer items-center border-b px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="https://discord.com/channels/842170679536517141/1051516534029291581">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiDiscordLine className="text-subtle mr-2 h-4 w-4" />
                  <div>Join Discord</div>
                </div>
              </Link>
            </Listbox.Option>
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="https://roadmap.inngest.com/roadmap">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiRoadMapLine className="text-subtle mr-2 h-4 w-4" />
                  <div>Inngest Roadmap</div>
                </div>
              </Link>
            </Listbox.Option>
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="/support">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <SystemStatusIcon status={status} className="mx-0 mr-2 h-3.5 w-3.5" />
                  <div>Status</div>
                </div>
              </Link>
            </Listbox.Option>
            <Listbox.Option
              className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              <Link href="https://roadmap.inngest.com/changelog">
                <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                  <RiArticleLine className="text-subtle mr-2 h-4 w-4" />
                  <div>Release Notes</div>
                </div>
              </Link>
            </Listbox.Option>
          </Listbox.Options>
        </div>
      </Listbox>
    </div>
  );
};
