'use client';

import { Fragment, useState } from 'react';
import { type Route } from 'next';
import Link from 'next/link';
import { usePathname, useRouter, useSelectedLayoutSegments } from 'next/navigation';
import { Listbox, Transition } from '@headlessui/react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import {
  RiCloudFill,
  RiCloudLine,
  RiExpandUpDownLine,
  RiLoopLeftLine,
  RiSettings3Line,
} from '@remixicon/react';

import { useEnvironments } from '@/queries';
import cn from '@/utils/cn';
import {
  EnvironmentType,
  getDefaultEnvironment,
  getLegacyTestMode,
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
  const segmentsWithoutRouteGroups = segments.filter(
    (segment) => !segment.startsWith('(') && !segment.endsWith(')')
  );
  const pathname = usePathname();

  // Accounts are not environment specific
  if (pathname.match(/^\/settings\//)) {
    return '/functions';
  }

  // Deploys should always move to the root resource level
  if (segmentsWithoutRouteGroups[0] === 'apps') {
    return '/apps';
  }
  // Manage paths, we drop the id at the end
  if (segmentsWithoutRouteGroups[0] === 'manage') {
    return '/' + segmentsWithoutRouteGroups.slice(0, 2).join('/');
  }

  // Logs are specific to a given environment, return to the function dashboard
  if (segmentsWithoutRouteGroups[0] === 'functions' && segmentsWithoutRouteGroups[2] === 'logs') {
    return '/' + segmentsWithoutRouteGroups.slice(0, 3).join('/');
  }

  if (segmentsWithoutRouteGroups.length === 0) {
    return '/functions'; // default if selected from /env
  }

  return '/' + segmentsWithoutRouteGroups.join('/');
};

const selectedName = (name: string, collapsed: boolean) => {
  switch (name) {
    case 'Production':
      return collapsed ? 'PR' : name;
    case 'Branch Environments':
      return collapsed ? 'BE' : name;
    default:
      return collapsed ? name.substring(0, 2).toUpperCase() : name;
  }
};

type EnvironmentSelectMenuProps = {
  activeEnv?: Environment;
  collapsed: boolean;
};

export default function EnvironmentSelectMenu({
  activeEnv,
  collapsed,
}: EnvironmentSelectMenuProps) {
  const router = useRouter();
  const [selected, setSelected] = useState<Environment | null>(null);
  const nextPathname = useSwitchablePathname();
  const [{ fetching, data: envs = [] }] = useEnvironments();

  if (fetching || !isNonEmptyArray(envs)) {
    return <Skeleton className={`h-8 ${collapsed ? 'w-8 px-1' : 'w-[146px] px-2'}`} />;
  }

  const defaultEnvironment = getDefaultEnvironment(envs);
  const legacyTestMode = getLegacyTestMode(envs);
  const mostRecentlyCreatedBranchEnvironments = getSortedBranchEnvironments(envs).slice(0, 5);
  const testEnvironments = getTestEnvironments(envs);

  if (selected === null && activeEnv) {
    setSelected(activeEnv);
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
        <div className="bg-canvasBase relative flex">
          <Listbox.Button
            className={`border-muted bg-canvasBase text-primary-intense hover:bg-canvasSubtle ${
              collapsed ? `w-8 px-1` : 'w-[146px] px-2'
            } m-0 h-8 overflow-hidden rounded border text-sm ${open && 'border-primary-intense'}`}
          >
            <div
              className={`flex flex-row items-center  ${
                collapsed ? 'justify-center' : 'justify-between'
              }`}
            >
              <span className="flex flex-row items-center ">
                {isBranchParentSelected ? (
                  <>
                    {!collapsed && <RiSettings3Line className="mr-2 h-4" />}
                    <span className="block truncate">
                      {selectedName('Branch Environments', collapsed)}
                    </span>
                  </>
                ) : selected ? (
                  <span className="block truncate">
                    {selected.type === EnvironmentType.BranchParent
                      ? selectedName('Branch Environments', collapsed)
                      : selectedName(selected.name, collapsed)}
                  </span>
                ) : (
                  <>
                    {!collapsed && <RiCloudLine className="mr-2 h-4 w-4" />}
                    <span className="block truncate">
                      {selectedName('All Environments', collapsed)}
                    </span>
                  </>
                )}
              </span>

              {!collapsed && (
                <RiExpandUpDownLine className="text-subtle h-4 w-4" aria-hidden="true" />
              )}
            </div>
          </Listbox.Button>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="bg-canvasBase border-subtle absolute top-10 z-[1000] w-[188px] divide-none rounded border shadow focus:outline-none">
              {defaultEnvironment !== null && <EnvironmentItem environment={defaultEnvironment} />}

              {legacyTestMode !== null && (
                <div>
                  <EnvironmentItem environment={legacyTestMode} name="Test mode" />
                </div>
              )}

              {testEnvironments.length > 0 &&
                testEnvironments.map((env) => <EnvironmentItem key={env.id} environment={env} />)}

              <div>
                <div className="bg-canvasBase text-disabled border-subtle flex h-10 cursor-not-allowed items-center gap-3  border-t px-3 text-xs font-normal transition-all">
                  Branch Environments
                </div>
                {mostRecentlyCreatedBranchEnvironments.length > 0 ? (
                  mostRecentlyCreatedBranchEnvironments.map((env) => (
                    <EnvironmentItem key={env.id} environment={env} variant="compact" />
                  ))
                ) : (
                  <Link
                    href="/env"
                    className="bg-canvasBase hover:bg-canvasSubtle text-muted flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal transition-all"
                  >
                    <RiLoopLeftLine className="h-3 w-3" />
                    Sync a branch
                  </Link>
                )}
              </div>

              <div>
                <Link
                  prefetch={true}
                  href="/env"
                  className="hover:bg-canvasSubtle text-muted flex h-10 cursor-pointer items-center gap-2 whitespace-nowrap px-3 text-[13px] font-normal transition-all"
                >
                  <RiCloudFill className="h-3 w-3" />
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
        'bg-canvasBase hover:bg-canvasSubtle text-muted flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal transition-all',
        variant === 'compact' && 'py-2'
      )}
    >
      <span className={cn('block h-1.5 w-1.5 shrink-0 rounded-full', statusColorClass)} />
      <span className="truncate">{name || environment.name}</span>
    </Listbox.Option>
  );
}
