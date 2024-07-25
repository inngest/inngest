'use client';

import type { ReactNode } from 'react';
import Link from 'next/link';
import { SignOutButton } from '@clerk/nextjs';
import { Listbox } from '@headlessui/react';
import {
  RiArrowLeftRightLine,
  RiBillLine,
  RiEqualizerLine,
  RiGroupLine,
  RiLogoutCircleLine,
  RiUserLine,
} from '@remixicon/react';

export const ProfileMenu = ({ children }: { children: ReactNode }) => {
  return (
    <Listbox>
      <Listbox.Button as="div" className="cursor-pointer ring-0">
        {children}
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase absolute -right-48 bottom-4 z-50 ml-8 w-[199px] rounded border shadow ring-0 focus:outline-none">
          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            <Link href="/settings/organization/organization-settings">
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiEqualizerLine className="text-subtle mr-2 h-4 w-4 " />
                <div>Your Organization</div>
              </div>
            </Link>
          </Listbox.Option>
          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            <Link href="/settings/organization">
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiGroupLine className="text-subtle mr-2 h-4 w-4" />
                <div>Members</div>
              </div>
            </Link>
          </Listbox.Option>
          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            <Link href="/settings/billing">
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiBillLine className="text-subtle mr-2 h-4 w-4" />
                <div>Billing</div>
              </div>
            </Link>
          </Listbox.Option>
          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle border-subtle flex h-12 cursor-pointer items-center border-b px-4 text-[13px]"
            value="eventKeys"
          >
            <Link href="/organization-list">
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiArrowLeftRightLine className="text-subtle mr-2 h-4 w-4" />
                <div>Switch Organization</div>
              </div>
            </Link>
          </Listbox.Option>

          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            <Link href="/settings/user">
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiUserLine className="text-subtle mr-2 h-4 w-4" />
                <div>Your Profile</div>
              </div>
            </Link>
          </Listbox.Option>
          <Listbox.Option
            className="text-subtle hover:bg-canvasSubtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
            value="eventKeys"
          >
            <SignOutButton>
              <div className="hover:bg-canvasSubtle flex flex-row items-center justify-start">
                <RiLogoutCircleLine className="text-subtle mr-2 h-4 w-4" />
                <div>Sign Out</div>
              </div>
            </SignOutButton>
          </Listbox.Option>
        </Listbox.Options>
      </div>
    </Listbox>
  );
};
