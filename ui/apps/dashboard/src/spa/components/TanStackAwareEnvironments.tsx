import { useState } from 'react';
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
import { useRouter } from '@tanstack/react-router';

import { useEnvironments } from '@/queries';
import {
  EnvironmentType,
  getDefaultEnvironment,
  getSortedBranchEnvironments,
  getTestEnvironments,
  type Environment,
} from '@/utils/environments';

type TanStackEnvironmentsProps = {
  activeEnv?: Environment;
  collapsed: boolean;
};

const useTanStackSwitchablePathname = (): string => {
  const router = useRouter();
  const currentPath = router.state.location.pathname;

  if (currentPath.includes('/tanstack/env/')) {
    const pathParts = currentPath.split('/');
    const envIndex = pathParts.findIndex((part) => part === 'env');
    if (envIndex !== -1 && pathParts[envIndex + 2]) {
      return '/' + pathParts.slice(envIndex + 2).join('/');
    }
  }

  return '/functions';
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

export default function TanStackAwareEnvironments({
  activeEnv,
  collapsed,
}: TanStackEnvironmentsProps) {
  const router = useRouter();
  const [selected, setSelected] = useState<Environment | null>(null);
  const nextPathname = useTanStackSwitchablePathname();
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

    router.navigate({
      to: `/env/$envSlug${nextPathname}` as any,
      params: { envSlug: env.slug } as any,
    });
  };

  const handleViewAllEnvironments = (e: React.MouseEvent) => {
    e.preventDefault();
    window.location.href = '/env';
  };

  const handleSyncBranch = (e: React.MouseEvent) => {
    e.preventDefault();
    window.location.href = '/env';
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

          <Listbox.Options className="bg-canvasBase border-subtle absolute top-10 z-50 max-h-[calc(100vh-8rem)] w-[188px] divide-none overflow-y-auto rounded border shadow focus:outline-none">
            {defaultEnvironment !== null && (
              <EnvironmentItem environment={defaultEnvironment} name={defaultEnvironment.name} />
            )}

            {testEnvironments.length > 0 &&
              testEnvironments.map((env) => (
                <EnvironmentItem key={env.id} environment={env} name={env.name} />
              ))}

            <div>
              <div className="bg-canvasBase text-disabled border-subtle flex h-[18px] cursor-not-allowed items-center gap-3 border-t px-3 py-4 text-xs font-normal">
                Branch Environments
              </div>
              {mostRecentlyCreatedBranchEnvironments.length > 0 ? (
                mostRecentlyCreatedBranchEnvironments.map((env) => (
                  <EnvironmentItem
                    key={env.id}
                    environment={env}
                    name={env.name}
                    variant="compact"
                  />
                ))
              ) : (
                <div
                  onClick={handleSyncBranch}
                  className="bg-canvasBase hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal"
                >
                  <RiLoopLeftLine className="h-3 w-3" />
                  Sync a branch
                </div>
              )}
            </div>

            <div>
              <div
                onClick={handleViewAllEnvironments}
                className="hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-3 whitespace-nowrap px-3 text-[13px] font-normal"
              >
                <RiCloudFill className="h-3 w-3" />
                View All Environments
              </div>
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
