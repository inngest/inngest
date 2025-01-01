'use client';

import { type Route } from 'next';
import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { RiMore2Line } from '@remixicon/react';

import Toaster from '@/components/Toaster';
import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironments } from '@/queries';
import { EnvironmentType } from '@/utils/environments';
import { EnvironmentArchiveButton } from './EnvironmentArchiveButton';
import EnvironmentListTable from './EnvironmentListTable';

export default function Environments() {
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

        <NextLink
          href={process.env.NEXT_PUBLIC_HOME_PATH as Route}
          className="bg-info hover:bg-info/80 mt-4 flex items-center justify-between rounded-lg px-4 py-2"
        >
          <h3 className="flex items-center gap-2 text-sm font-medium tracking-wide">
            <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
            Production
          </h3>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button kind="secondary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
            </DropdownMenuTrigger>
            <DropdownMenuContent className="z-50">
              <DropdownMenuItem onSelect={() => router.push('/env/production/manage')}>
                Manage
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => router.push('/env/production/apps')}>
                Go to apps
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </NextLink>

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
                  <DropdownMenuContent className="z-50">
                    <DropdownMenuItem
                      onSelect={() => router.push(`/env/${branchParent?.slug}/manage`)}
                    >
                      Manage
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      className="text-success"
                      onSelect={() => router.push(`/env/${branchParent?.slug || 'branch'}/apps`)}
                    >
                      Sync new app
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            <div className=" border-subtle mt-8 overflow-hidden rounded-lg border">
              <EnvironmentListTable envs={branches} />
            </div>
          </div>
        )}

        {customEnvs.length > 0 && (
          <div className="border-subtle border-t pt-8">
            <div className="mb-4 flex w-full items-center justify-between">
              <h2 className="text-lg font-medium">Custom Environments</h2>
              <Button href="create-environment" kind="primary" label="Create environment" />
            </div>
            {customEnvs.map((env) => (
              <div key={env.id}>
                <NextLink
                  href={`/env/${env.slug}/functions` as Route}
                  className="hover:bg-canvasMuted border-subtle bg-canvasBase mt-4 flex cursor-pointer items-center justify-between rounded-lg border px-4 py-1.5"
                >
                  <h3 className="flex items-center gap-2 text-sm font-medium">
                    <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
                    {env.name}
                  </h3>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        kind="secondary"
                        appearance="outlined"
                        size="medium"
                        icon={<RiMore2Line />}
                      />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent className="z-50">
                      <DropdownMenuItem onSelect={() => router.push(`/env/${env.slug}/manage`)}>
                        Manage
                      </DropdownMenuItem>
                      <DropdownMenuItem className="text-success" asChild>
                        <EnvironmentArchiveButton env={env} />
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className="text-success"
                        onSelect={() => router.push(`/env/${env.slug}/apps`)}
                      >
                        Go to apps
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </NextLink>
              </div>
            ))}
          </div>
        )}
      </div>

      <Toaster />
    </>
  );
}
