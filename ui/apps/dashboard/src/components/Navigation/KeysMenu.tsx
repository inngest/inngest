'use client';

import NextLink from 'next/link';
import { Listbox } from '@headlessui/react';
import { Button } from '@inngest/components/Button';
import { RiEqualizer2Line } from '@remixicon/react';

import type { Environment as EnvType } from '@/utils/environments';

export default function KeysMenu({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  return (
    <Listbox>
      <Listbox.Button as="div" className="group">
        <Button
          kind="secondary"
          appearance={collapsed ? 'ghost' : 'outlined'}
          size="medium"
          icon={<RiEqualizer2Line className="fill-subtle w-[18px]" />}
          className="group-data-[headlessui-state=open]:border-primary-moderate"
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase border-muted shadow-primary absolute left-0 z-50 ml-1 w-[137px] gap-y-0.5 rounded border">
          <NextLink href={`/env/${activeEnv.slug}/manage/keys`} prefetch={true}>
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle mx-2 mt-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="eventKeys"
            >
              Event keys
            </Listbox.Option>
          </NextLink>
          <NextLink href={`/env/${activeEnv.slug}/manage/signing-key`} prefetch={true}>
            <Listbox.Option
              className="text-muted hover:bg-canvasSubtle m-2 flex h-8 cursor-pointer items-center px-2 text-[13px]"
              value="signingKeys"
            >
              Signing keys
            </Listbox.Option>
          </NextLink>
        </Listbox.Options>
      </div>
    </Listbox>
  );
}
