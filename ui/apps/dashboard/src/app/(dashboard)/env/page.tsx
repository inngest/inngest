import { type Route } from 'next';
import Link from 'next/link';
import { ExclamationTriangleIcon } from '@heroicons/react/20/solid';

import Button from '@/components/Button';
import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';
import getAllEnvironments from '@/queries/server-only/getAllEnvironments';
import { EnvironmentType, LEGACY_TEST_MODE_NAME } from '@/utils/environments';
import EnvironmentListTable from './EnvironmentListTable';

export default async function Envs() {
  const environments = await getAllEnvironments();

  // Break the environments into different groups
  const legacyTestMode = environments?.find(
    (env) => env.type === EnvironmentType.Test && env.name === LEGACY_TEST_MODE_NAME
  );
  const branchParent = environments?.find((env) => env.type === EnvironmentType.BranchParent);
  const branches = environments?.filter((env) => env.type === EnvironmentType.BranchChild);
  const otherTestEnvs = environments?.filter(
    (env) => env.type === EnvironmentType.Test && env.name !== LEGACY_TEST_MODE_NAME
  );

  return (
    <>
      <div className="flex h-full flex-col">
        <AppNavigation environmentSlug="all" />
        <div className="overflow-y-scroll">
          <div className="mx-auto w-full max-w-[860px] px-12 py-16">
            <div className="mb-4 flex w-full items-center  justify-between">
              <h2 className="text-lg font-medium text-slate-800">Production Environment</h2>
              <div className="flex items-center gap-2">
                <Button href="/env/production/manage" variant="secondary" size="base">
                  Manage
                </Button>
                <Button
                  href={'/env/production/deploys?intent=deploy-modal' as Route}
                  variant="primary"
                  size="base"
                >
                  Deploy
                </Button>
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
                <span className="block h-2 w-2 rounded-full bg-teal-400" />
                Production
              </h3>
              {/* <div className="flex items-center gap-2">
          <span className="text-white text-sm font-regular tracking-wide">10 minutes ago</span>
          <CheckCircleIcon className="w-4 h-4 text-teal-400" />
        </div> */}
            </Link>

            {Boolean(legacyTestMode) && (
              <div className="mt-12 border-t border-slate-100 pt-8">
                <div className="mb-4 flex w-full items-center  justify-between">
                  <h2 className="text-lg font-medium text-slate-800">Test Mode</h2>
                  <div className="flex items-center gap-2">
                    <Button href="/env/test/manage" variant="secondary" size="base">
                      Manage
                    </Button>
                    <Button
                      href="/env/test/deploys?intent=deploy-modal"
                      variant="primary"
                      size="base"
                    >
                      Deploy
                    </Button>
                  </div>
                </div>
                <Link
                  href={'/env/test/functions' as Route}
                  className="mt-8 flex cursor-pointer items-center justify-between rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm hover:bg-slate-100/60"
                >
                  <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-800">
                    <span className="block h-2 w-2 rounded-full bg-teal-500" />
                    Test
                  </h3>
                  {/* <div className="flex justify-end items-center gap-2">
              <span className="text-slate-600 text-sm font-medium">Just now</span>
              <CheckCircleIcon className="w-4 h-4 text-teal-500" />
            </div> */}
                </Link>
                <p className="mt-4 text-sm text-amber-600">
                  <ExclamationTriangleIcon className="mr-1 inline-block h-4 w-4 text-amber-500" />
                  Test mode is a legacy environment
                  {/*Read the{' '}
          <Link className="font-semibold underline text-amber-500" href="/">
            upgrade guide
          </Link>{' '}
  to learn how to migrate.*/}
                </p>
              </div>
            )}

            {Boolean(branchParent) && (
              <div className="mt-12 border-t border-slate-100 pt-8">
                <div className="mb-8 flex w-full items-center justify-between">
                  <h2 className="text-lg font-medium text-slate-800">Branch Environments</h2>
                  <div className="flex items-center gap-2">
                    <Button
                      href={`/env/${branchParent?.slug}/manage`}
                      variant="secondary"
                      size="base"
                    >
                      Manage
                    </Button>
                    {/* Here we don't link to the modal since the /deploy empty state has more info on branch envs */}
                    <Button
                      href={`/env/${branchParent?.slug || 'branch'}/deploys` as Route}
                      variant="primary"
                      size="base"
                    >
                      Deploy
                    </Button>
                  </div>
                </div>

                <div className=" mb-20 mt-8 overflow-hidden rounded-lg border border-slate-200 shadow-sm">
                  <EnvironmentListTable envs={branches} />
                </div>
              </div>
            )}

            {otherTestEnvs?.length > 0 &&
              otherTestEnvs.map((env) => (
                <div key={env.id} className="mt-12 border-t border-slate-100 pt-8">
                  <div className="mb-4 flex w-full items-center  justify-between">
                    <h2 className="text-lg font-medium text-slate-800">{env.name}</h2>
                    <div className="flex items-center gap-2">
                      <Button href={`/env/${env.slug}/manage`} variant="secondary" size="base">
                        Manage
                      </Button>
                      <Button
                        href={`/env/${env.slug}/deploys?intent=deploy-modal` as Route}
                        variant="primary"
                        size="base"
                      >
                        Deploy
                      </Button>
                    </div>
                  </div>
                  <Link
                    href={`/env/${env.slug}/functions` as Route}
                    className="mt-8 flex cursor-pointer items-center justify-between rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm hover:bg-slate-100/60"
                  >
                    <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-800">
                      <span className="block h-2 w-2 rounded-full bg-teal-500" />
                      {env.name}
                    </h3>
                    {/* <div className="flex justify-end items-center gap-2">
                    <span className="text-slate-600 text-sm font-medium">Just now</span>
                    <CheckCircleIcon className="w-4 h-4 text-teal-500" />
                  </div> */}
                  </Link>
                </div>
              ))}
          </div>
        </div>
      </div>
      <Toaster />
    </>
  );
}
