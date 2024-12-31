'use client';

import { type Route } from 'next';
import NextLink from 'next/link';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';

import Toaster from '@/components/Toaster';
import LoadingIcon from '@/icons/LoadingIcon';
import { useEnvironments } from '@/queries';
import { EnvironmentType } from '@/utils/environments';
import { EnvironmentArchiveButton } from './EnvironmentArchiveButton';
import EnvironmentListTable from './EnvironmentListTable';

export default function Environments() {
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
          <div className="flex items-center gap-2">
            <Button href="/env/production/manage" appearance="outlined" label="Manage" />
            <Button href={`/env/production/apps` as Route} kind="primary" label="Go to apps" />
          </div>
        </div>
        <p className="text-muted mt-2 max-w-[400px] text-sm">
          This is where you&apos;ll deploy all of your production apps.
        </p>

        <NextLink
          href={process.env.NEXT_PUBLIC_HOME_PATH as Route}
          className="bg-surfaceMuted hover:bg-surfaceMuted/80 mt-4 flex items-center justify-between rounded-lg px-4 py-4"
        >
          <h3 className="flex items-center gap-2 text-sm font-medium tracking-wide">
            <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
            Production
          </h3>
        </NextLink>

        {Boolean(branchParent) && (
          <div className="border-subtle mt-12 border-t pt-8">
            <div className="mb-8 flex w-full items-center justify-between">
              <h2 className="text-lg font-medium">Branch Environments</h2>
              <div className="flex items-center gap-2">
                <Button
                  href={`/env/${branchParent?.slug}/manage`}
                  appearance="outlined"
                  label="Manage"
                />

                {/* Here we don't link to the modal since the /deploy empty state has more info on branch envs */}
                <Button
                  href={`/env/${branchParent?.slug || 'branch'}/apps` as Route}
                  kind="primary"
                  label="Sync new app"
                />
              </div>
            </div>

            <div className=" border-subtle mb-20 mt-8 overflow-hidden rounded-lg border">
              <EnvironmentListTable envs={branches} />
            </div>
          </div>
        )}

        {customEnvs.length > 0 &&
          customEnvs.map((env) => (
            <div key={env.id} className="border-subtle mt-12 border-t pt-8">
              <div className="mb-4 flex w-full items-center  justify-between">
                <h2 className="text-lg font-medium">{env.name}</h2>
                <div className="flex items-center gap-2">
                  <Button href={`/env/${env.slug}/manage`} appearance="outlined" label="Manage" />

                  <EnvironmentArchiveButton env={env} />

                  <Button
                    href={`/env/${env.slug}/apps` as Route}
                    kind="primary"
                    label="Go to apps"
                  />
                </div>
              </div>
              <NextLink
                href={`/env/${env.slug}/functions` as Route}
                className="hover:bg-canvasMuted border-subtle bg-canvasBase mt-8 flex cursor-pointer items-center justify-between rounded-lg border px-4 py-3"
              >
                <h3 className="flex items-center gap-2 text-sm font-semibold">
                  <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
                  {env.name}
                </h3>
              </NextLink>
            </div>
          ))}

        <div className="border-subtle mt-12 border-t pt-8">
          <div className="mb-4 flex w-full items-center  justify-between">
            <h2 className="text-lg font-medium">Create an environment</h2>
            <div className="flex items-center gap-2">
              <Button href="create-environment" kind="primary" label="Create environment" />
            </div>
          </div>
          <p className="text-muted mt-2 text-sm">
            Create a shared, non-production environment like staging, QA, or canary.{' '}
            <Link
              size="small"
              className="inline-flex"
              href="https://www.inngest.com/docs/platform/environments#custom-environments"
            >
              Read the docs to learn more
            </Link>
          </p>
        </div>
      </div>

      <Toaster />
    </>
  );
}
