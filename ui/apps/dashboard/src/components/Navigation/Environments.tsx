'use client';

import { useState } from 'react';
import { type Route } from 'next';
import NextLink from 'next/link';
import { usePathname, useRouter, useSelectedLayoutSegments } from 'next/navigation';
import { Listbox } from '@headlessui/react';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiCloudFill,
  RiCloudLine,
  RiErrorWarningLine,
  RiExpandUpDownLine,
  RiLoopLeftLine,
} from '@remixicon/react';

import { useEnvironments } from '@/queries';
import {
  EnvironmentType,
  getDefaultEnvironment,
  getSortedBranchEnvironments,
  getTestEnvironments,
  type Environment,
} from '@/utils/environments';

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

  if (pathname.match(/^\/billing\/.+$/)) {
    return '/apps';
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

const SelectedDisplay = ({
  selected,
  collapsed,
}: {
  selected: Environment | null;
  collapsed: boolean;
}) => (
  <span className={`flex flex-row items-center ${collapsed ? '' : 'min-w-0 truncate'}`}>
    {selected ? (
      <span className="block">
        {selected.type === EnvironmentType.BranchParent
          ? selectedName('Branch Environments', collapsed)
          : selectedName(selected.name, collapsed)}
      </span>
    ) : (
      <>
        {!collapsed && <RiCloudLine className="mr-2 h-4 w-4" />}
        <span className="block">{selectedName('All Environments', collapsed)}</span>
      </>
    )}
  </span>
);

const tooltip = (selected: Environment | null) =>
  !selected
    ? 'All Environments'
    : selected.type === EnvironmentType.BranchParent
    ? 'Branch Environments'
    : selected.name;

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
  const [{ data: envs = [], error }] = useEnvironments();

  if (error) {
    console.error('error fetching envs', error);
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="bg-error text-error flex h-8 w-full items-center justify-start gap-x-2 rounded px-2">
            <RiErrorWarningLine className="w-4" />
            {!collapsed && <div>Env Error</div>}
          </div>
        </TooltipTrigger>
        <TooltipContent side="right" className="text-error bg-error rounded text-xs">
          Error loading environments. Please try again or contact support if the issue does not
          resolve.
        </TooltipContent>
      </Tooltip>
    );
  }

  const defaultEnvironment = getDefaultEnvironment(envs);
  const includeArchived = false;
  const mostRecentlyCreatedBranchEnvironments = getSortedBranchEnvironments(
    envs,
    includeArchived
  ).slice(0, 5);
  const testEnvironments = getTestEnvironments(envs, includeArchived);

  if (selected === null && activeEnv) {
    setSelected(activeEnv);
  }

  const onSelect = (env: Environment) => {
    setSelected(env);

    // When switching environments, use the switchable pathname
    router.push(`/env/${env.slug}${nextPathname}` as Route);
  };

  return (
    <Listbox value={selected} onChange={onSelect}>
      {({ open }) => (
        <div className="bg-canvasBase relative flex">
          <OptionalTooltip tooltip={collapsed && tooltip(selected)}>
            <Listbox.Button
              className={`border-muted bg-canvasBase text-primary-intense hover:bg-canvasSubtle px-2 ${
                collapsed ? `w-8` : !activeEnv ? 'w-[196px]' : 'w-[158px]'
              } h-8 overflow-hidden rounded border text-sm ${open && 'border-primary-intense'}`}
            >
              <div
                className={`flex flex-row items-center  ${
                  collapsed ? 'justify-center' : 'justify-between'
                }`}
              >
                <SelectedDisplay selected={selected} collapsed={collapsed} />
                {!collapsed && (
                  <RiExpandUpDownLine className="text-muted h-4 w-4" aria-hidden="true" />
                )}
              </div>
            </Listbox.Button>
          </OptionalTooltip>

          <Listbox.Options className="bg-canvasBase border-subtle overflow-y-truncate absolute top-10 z-50 w-[188px] divide-none rounded border shadow focus:outline-none">
            {defaultEnvironment !== null && <EnvironmentItem environment={defaultEnvironment} />}

            {testEnvironments.length > 0 &&
              testEnvironments.map((env) => <EnvironmentItem key={env.id} environment={env} />)}

            <div>
              <div className="bg-canvasBase text-disabled border-subtle flex h-[18px] cursor-not-allowed items-center gap-3 border-t px-3 py-4 text-xs font-normal">
                Branch Environments
              </div>
              {mostRecentlyCreatedBranchEnvironments.length > 0 ? (
                mostRecentlyCreatedBranchEnvironments.map((env) => (
                  <EnvironmentItem key={env.id} environment={env} variant="compact" />
                ))
              ) : (
                <NextLink
                  href="/env"
                  className="bg-canvasBase hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal"
                >
                  <RiLoopLeftLine className="h-3 w-3" />
                  Sync a branch
                </NextLink>
              )}
            </div>

            <div>
              <NextLink
                prefetch={true}
                href="/env"
                className="hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-2 whitespace-nowrap px-3 text-[13px] font-normal"
              >
                <RiCloudFill className="h-3 w-3" />
                View All Environments
              </NextLink>
            </div>
          </Listbox.Options>
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
    statusColorClass = 'bg-surfaceMuted';
  } else {
    statusColorClass = 'bg-primary-moderate';
  }

  return (
    <Listbox.Option
      key={environment.id}
      value={environment}
      className={cn(
        'bg-canvasBase hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal',
        variant === 'compact' && 'py-2'
      )}
    >
      <span className={cn('block h-1.5 w-1.5 shrink-0 rounded-full', statusColorClass)} />
      <span className="truncate">{name || environment.name}</span>
    </Listbox.Option>
  );
}
