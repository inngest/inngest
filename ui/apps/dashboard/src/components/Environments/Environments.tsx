'use client';

import { type Route } from 'next';
import Link from 'next/link';
import { Button } from '@inngest/components/Button';
import { Link as InngestLink } from '@inngest/components/Link';

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
          <h2 className="text-lg font-medium text-slate-800">Production Environment</h2>
          <div className="flex items-center gap-2">
            <Button href="/env/production/manage" appearance="outlined" label="Manage" />
            <Button href={`/env/production/apps` as Route} kind="primary" label="Go To Apps" />
          </div>
        </div>
        <p className="mt-2 max-w-[400px] text-sm font-medium text-slate-600">
          This is where you&apos;ll deploy all of your production apps.
        </p>

        <Link
          href={process.env.NEXT_PUBLIC_HOME_PATH as Route}
          className="to-slate-940 mt-4 flex items-center justify-between rounded-lg bg-slate-900 bg-gradient-to-br from-slate-800 px-4 py-4 hover:bg-slate-800 hover:from-slate-700 hover:to-slate-900"
        >
          <h3 className="flex items-center gap-2 text-sm font-medium tracking-wide text-white">
            <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
            Production
          </h3>
        </Link>

        {Boolean(branchParent) && (
          <div className="mt-12 border-t border-slate-100 pt-8">
            <div className="mb-8 flex w-full items-center justify-between">
              <h2 className="text-lg font-medium text-slate-800">Branch Environments</h2>
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
                  label="Sync New App"
                />
              </div>
            </div>

            <div className=" mb-20 mt-8 overflow-hidden rounded-lg border border-slate-200 shadow-sm">
              <EnvironmentListTable envs={branches} />
            </div>
          </div>
        )}

        {customEnvs.length > 0 &&
          customEnvs.map((env) => (
            <div key={env.id} className="mt-12 border-t border-slate-100 pt-8">
              <div className="mb-4 flex w-full items-center  justify-between">
                <h2 className="text-lg font-medium text-slate-800">{env.name}</h2>
                <div className="flex items-center gap-2">
                  <Button href={`/env/${env.slug}/manage`} appearance="outlined" label="Manage" />

                  <EnvironmentArchiveButton env={env} />

                  <Button
                    href={`/env/${env.slug}/apps` as Route}
                    kind="primary"
                    label="Go To Apps"
                  />
                </div>
              </div>
              <Link
                href={`/env/${env.slug}/functions` as Route}
                className="mt-8 flex cursor-pointer items-center justify-between rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm hover:bg-slate-100/60"
              >
                <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-800">
                  <span className="bg-primary-moderate block h-2 w-2 rounded-full" />
                  {env.name}
                </h3>
              </Link>
            </div>
          ))}

        <div className="mt-12 border-t border-slate-100 pt-8">
          <div className="mb-4 flex w-full items-center  justify-between">
            <h2 className="text-lg font-medium text-slate-800">Create an environment</h2>
            <div className="flex items-center gap-2">
              <Button href="create-environment" kind="primary" label="Create environment" />
            </div>
          </div>
          <p className="mt-2 text-sm font-medium text-slate-600">
            Create a shared, non-production environment like staging, QA, or canary.{' '}
            <InngestLink
              className="inline-flex"
              href="https://www.inngest.com/docs/platform/environments#custom-environments"
            >
              Read the docs to learn more
            </InngestLink>
          </p>
        </div>
      </div>

      <Toaster />
    </>
  );
}
