'use client';

import { Fragment, useState } from 'react';
import { type Route } from 'next';
import Link from 'next/link';
import { usePathname, useRouter, useSelectedLayoutSegments } from 'next/navigation';
import { Listbox, Transition } from '@headlessui/react';
import {
  ChevronUpDownIcon,
  CloudIcon,
  Cog6ToothIcon,
  RocketLaunchIcon,
} from '@heroicons/react/20/solid';

import { useEnvironments } from '@/queries/environments';
import cn from '@/utils/cn';
import {
  EnvironmentType,
  getActiveEnvironment,
  getLegacyTestMode,
  getProductionEnvironment,
  getSortedBranchEnvironments,
  getTestEnvironments,
  type Environment,
} from '@/utils/environments';
import isNonEmptyArray from '@/utils/isNonEmptyArray';

// Some URLs cannot just swap between environments,
// we need to redirect to a less specific resource URL that is shared across environments
// for the user to switch context correctly
const useSwitchablePathname = (): string => {
  const segments = useSelectedLayoutSegments();
  const pathname = usePathname();

  // Accounts are not environment specific
  if (pathname.match(/^\/settings\//)) {
    return '/functions';
  }

  // Deploys should always move to the root resource level
  if (segments[0] === 'deploys') {
    return '/deploys';
  }
  // Manage paths, we drop the id at the end
  if (segments[0] === 'manage') {
    return '/' + segments.slice(0, 2).join('/');
  }

  // Logs are specific to a given environment, return to the function dashboard
  if (segments[0] === 'functions' && segments[2] === 'logs') {
    return '/' + segments.slice(0, 3).join('/');
  }

  if (segments.length === 0) {
    return '/functions'; // default if selected from /env
  }

  return '/' + segments.join('/');
};

type EnvironmentSelectMenuProps = {
  environmentSlug: string;
};

export default function EnvironmentSelectMenu({ environmentSlug }: EnvironmentSelectMenuProps) {
  const router = useRouter();
  const [{ data: environments, fetching }] = useEnvironments();
  const [selected, setSelected] = useState<Environment | null>(null);
  const nextPathname = useSwitchablePathname();

  if (fetching || !environments || !isNonEmptyArray(environments)) {
    return (
      <div className="relative self-stretch border-x border-slate-800">
        <div className="font-regular flex h-full w-[180px] items-center gap-0.5 py-1.5 pl-4 pr-4  text-sm tracking-wide text-white hover:bg-slate-800">
          <span className="text-shadow pr-4 text-sm font-medium text-white">Loading...</span>
          <span className="pointer-events-none absolute inset-y-0 right-2 flex items-center">
            <ChevronUpDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </span>
        </div>
      </div>
    );
  }

  const activeEnvironment = getActiveEnvironment(environments, environmentSlug);
  const productionEnvironment = getProductionEnvironment(environments);
  const legacyTestMode = getLegacyTestMode(environments);
  const mostRecentlyCreatedBranchEnvironments = getSortedBranchEnvironments(environments).slice(
    0,
    5
  );
  const testEnvironments = getTestEnvironments(environments);

  if (selected === null && activeEnvironment) {
    setSelected(activeEnvironment);
  }
  const isBranchParentSelected = selected?.type === EnvironmentType.BranchParent;

  const onSelect = (env: Environment) => {
    setSelected(env);

    // When switching environments, use the switchable pathname
    router.push(`/env/${env.slug}${nextPathname}` as Route);
  };

  return (
    <Listbox value={selected} onChange={onSelect}>
      {({ open }) => (
        <div className="relative self-stretch border-x border-slate-800">
          <Listbox.Button className="font-regular flex h-full w-[180px] items-center gap-0.5 py-1.5 pl-4 pr-4 text-sm  tracking-wide text-white transition-all hover:bg-slate-800">
            <span className="flex max-w-full items-center pr-4">
              {isBranchParentSelected ? (
                <>
                  <Cog6ToothIcon className="mr-2 h-4" />
                  <span className="block truncate">Branch Environments</span>
                </>
              ) : selected ? (
                <>
                  <span className="mr-2 h-2 w-2 flex-shrink-0 rounded-full bg-cyan-500" />
                  <span className="block truncate">
                    {selected?.type === EnvironmentType.BranchParent
                      ? 'Branch Environments'
                      : selected?.name}
                  </span>
                </>
              ) : (
                <>
                  <CloudIcon className="mr-2 h-4" />
                  {/* <span className="w-2 h-2 rounded-full bg-cyan-500 mr-2 flex-shrink-0" /> */}
                  <span className="block truncate">All Environments</span>
                </>
              )}
            </span>
            <span className="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <ChevronUpDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
            </span>
          </Listbox.Button>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="bg-slate-1000/95 absolute left-0 z-10 mt-2 w-[280px] origin-top-right divide-y divide-dashed divide-slate-700 rounded-md text-sm backdrop-blur focus:outline-none">
              {/* <div className="py-1 pl-4 pr-1 flex gap-1 items-center border-b border-slate-700">
                <MagnifyingGlassIcon className="h-3 text-white" />
                <input
                  type="text"
                  placeholder="Find Environment..."
                  className="bg-transparent hover:bg-slate-900 focus:bg-slate-900 focus:outline-none focus:placeholder:text-slate-500 rounded w-full text-sm py-1 px-2.5 placeholder-slate-300 text-sm text-white"
                />
              </div> */}

              {productionEnvironment !== null && (
                <EnvironmentItem environment={productionEnvironment} />
              )}

              {legacyTestMode !== null && (
                <div>
                  <EnvironmentItem environment={legacyTestMode} name="Test mode" />
                  {/* <div className="px-3.5 pb-3 flex items-center gap-2 text-yellow-300">
                    <ExclamationTriangleIcon className="h-4" />
                    <span className="text-xs font-medium">Test Mode is a legacy environment</span>
                  </div> */}
                </div>
              )}

              {testEnvironments.length > 0 &&
                testEnvironments.map((env) => <EnvironmentItem key={env.id} environment={env} />)}

              <div>
                <div className="px-4 py-3 pb-1 font-medium text-white">Branch Environments</div>
                {mostRecentlyCreatedBranchEnvironments.length > 0 ? (
                  mostRecentlyCreatedBranchEnvironments.map((env) => (
                    <EnvironmentItem key={env.id} environment={env} variant="compact" />
                  ))
                ) : (
                  <Link
                    href="/env"
                    className="block flex items-center gap-2.5 px-3.5 py-3 font-medium text-slate-300 transition-all hover:bg-slate-800 hover:text-white"
                  >
                    <RocketLaunchIcon className="h-3" />
                    Deploy a branch
                  </Link>
                )}
              </div>

              <div className="flex items-center">
                <Link
                  href="/env"
                  className="flex w-full cursor-pointer items-center gap-2 truncate rounded px-3.5 py-3 text-sm text-slate-50 transition-all hover:bg-slate-700 hover:text-white"
                >
                  <CloudIcon className="h-3" />
                  View All Environments
                </Link>
              </div>
            </Listbox.Options>
          </Transition>
        </div>
      )}
    </Listbox>
  );
}

function EnvironmentItem({
  environment,
  name,
  variant = 'normal',
}: {
  environment: Environment;
  name?: string;
  variant?: 'compact' | 'normal';
}) {
  let statusColorClass: string;
  if (environment.isArchived) {
    statusColorClass = 'bg-slate-300';
  } else {
    statusColorClass = 'bg-teal-500';
  }

  return (
    <Listbox.Option
      key={environment.id}
      value={environment}
      className={cn(
        'flex cursor-pointer items-center gap-3 rounded px-4 py-3 text-sm font-medium text-slate-300 transition-all hover:bg-slate-800 hover:text-white',
        variant === 'compact' && 'py-2'
      )}
    >
      <span className={cn('block h-2 w-2 shrink-0 rounded-full', statusColorClass)} />
      <span className="truncate">{name || environment.name}</span>
    </Listbox.Option>
  );
}
