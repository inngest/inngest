'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { cn } from '@inngest/components/utils/classNames';
import { RiAddLine, RiMore2Line, RiSettingsLine } from '@remixicon/react';

import Toaster from '@/components/Toaster';
import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironments } from '@/queries';
import { EnvironmentType } from '@/utils/environments';
import { EnvironmentArchiveDropdownItem } from './EnvironmentArchiveDropdownItem';
import EnvironmentListTable from './EnvironmentListTable';

export default function Environments() {
  const [openCustomEnvDropdownId, setOpenCustomEnvDropdownId] = useState<string | null>(null);
  const router = useRouter();
  const [{ data: envs = [], fetching }] = useEnvironments();

  const branchParent = envs.find((env) => env.type === EnvironmentType.BranchParent);
  const branches = envs.filter((env) => env.type === EnvironmentType.BranchChild);
  const customEnvs = envs.filter((env) => env.type === EnvironmentType.Test);

  if (fetching) {
    return (
      <div className="my-16 flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  return (
    <>
      <div className="mx-auto w-full max-w-[860px] px-12 py-16">
        <div className="mb-4 flex w-full items-center  justify-between">
          <h2 className="text-lg font-medium">Production Environment</h2>
        </div>
        <p className="text-muted mt-2 max-w-[400px] text-sm">
          This is where you&apos;ll deploy all of your production apps.
        </p>

        <div className="bg-info mt-4 flex items-center justify-between rounded-md px-4 py-2">
          <h3 className="flex items-center gap-2 text-sm font-medium tracking-wide">
            <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
            Production
          </h3>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button kind="secondary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onSelect={() => router.push('/env/production/manage')}>
                <RiSettingsLine className="h-4 w-4" />
                Manage
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => router.push('/env/production/apps')}>
                <AppsIcon className="h-4 w-4" />
                Go to apps
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {Boolean(branchParent) && (
          <div className="border-subtle my-12 border-t pt-8">
            <div className="mb-8 flex w-full items-center justify-between">
              <h2 className="text-lg font-medium">Branch Environments</h2>
              <div className="flex items-center gap-2">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      kind="secondary"
                      appearance="outlined"
                      size="medium"
                      icon={<RiMore2Line />}
                    />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem
                      onSelect={() => router.push(`/env/${branchParent?.slug}/manage`)}
                    >
                      <RiSettingsLine className="h-4 w-4" />
                      Manage
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      className="text-success"
                      onSelect={() => router.push(`/env/${branchParent?.slug || 'branch'}/apps`)}
                    >
                      <RiAddLine className="h-4 w-4" />
                      Sync new app
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            <div className=" border-subtle mt-8 overflow-hidden rounded-md border">
              <EnvironmentListTable envs={branches} />
            </div>
          </div>
        )}

        <div className="border-subtle border-t pt-8">
          <div className="mb-4 flex w-full items-center justify-between">
            <h2 className="text-lg font-medium">Custom Environments</h2>
            <Button href="create-environment" kind="primary" label="Create environment" />
          </div>
          {customEnvs.length > 0 ? (
            customEnvs.map((env) => (
              <div
                key={env.id}
                className="border-subtle bg-canvasBase mt-4 flex items-center justify-between rounded-md border px-4 py-1.5"
              >
                <h3 className="flex items-center gap-2 text-sm font-medium">
                  <span
                    className={cn(
                      'block h-2 w-2 rounded-full',
                      env.isArchived ? 'bg-surfaceMuted' : 'bg-primary-moderate '
                    )}
                  />
                  {env.name}
                </h3>
                <DropdownMenu
                  modal
                  open={openCustomEnvDropdownId === env.id}
                  onOpenChange={(open) => setOpenCustomEnvDropdownId(open ? env.id : null)}
                >
                  <DropdownMenuTrigger asChild>
                    <Button
                      kind="secondary"
                      appearance="outlined"
                      size="medium"
                      icon={<RiMore2Line />}
                    />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem onSelect={() => router.push(`/env/${env.slug}/manage`)}>
                      <RiSettingsLine className="h-4 w-4" />
                      Manage
                    </DropdownMenuItem>

                    <DropdownMenuItem onSelect={() => router.push(`/env/${env.slug}/apps`)}>
                      <AppsIcon className="h-4 w-4" />
                      Go to apps
                    </DropdownMenuItem>
                    <EnvironmentArchiveDropdownItem
                      env={env}
                      onClose={() => setOpenCustomEnvDropdownId(null)}
                    />
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            ))
          ) : (
            <p className="text-basis py-3 text-center text-sm">No custom environments yet</p>
          )}
        </div>
      </div>

      <Toaster />
    </>
  );
}
