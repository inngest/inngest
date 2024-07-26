'use client';

import { Listbox } from '@headlessui/react';
import { HelpIcon } from '@inngest/components/icons/sections/Help';

import { MenuItem } from './MenuItem';

export const Help = ({ collapsed }: { collapsed: boolean }) => (
  <div className="m-2.5">
    <Listbox>
      <Listbox.Button as="div" className="ring-0">
        <MenuItem
          collapsed={collapsed}
          text="Help and Feedback"
          icon={<HelpIcon className="w-5" />}
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase absolute -right-48 bottom-0 z-50 ml-8 w-[199px] rounded border shadow ring-0 focus:outline-none">
          <Listbox.Option
            className="text-subtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            coming soon...
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  </div>
);
