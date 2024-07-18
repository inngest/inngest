'use client';

import Link from 'next/link';
import { Listbox } from '@headlessui/react';
import { NewButton } from '@inngest/components/Button';
import { RiEqualizer2Line } from '@remixicon/react';

import type { Environment as EnvType } from '@/utils/environments';

export default function KeysMenu({ activeEnv }: { activeEnv: EnvType }) {
  return (
    <Listbox value={true}>
      <Listbox.Button as="div">
        <NewButton
          kind="secondary"
          appearance="outlined"
          size="medium"
          icon={<RiEqualizer2Line className="fill-subtle" />}
          className="ml-2.5"
        />
      </Listbox.Button>
      <div className="relative">
        <Listbox.Options className="bg-canvasBase absolute left-0 z-50 ml-1 w-[137px] rounded border shadow">
          <Link href={`/env/${activeEnv.slug}/manage/keys`} prefetch={true}>
            <Listbox.Option
              className="text-subtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="eventKeys"
            >
              Event keys
            </Listbox.Option>
          </Link>
          <Link href={`/env/${activeEnv.slug}/manage/signing-key`} prefetch={true}>
            <Listbox.Option
              className="text-subtle flex h-12 cursor-pointer items-center px-4 text-[13px]"
              value="signingKeys"
            >
              Signing Keys
            </Listbox.Option>
          </Link>
        </Listbox.Options>
      </div>
    </Listbox>
  );
}
