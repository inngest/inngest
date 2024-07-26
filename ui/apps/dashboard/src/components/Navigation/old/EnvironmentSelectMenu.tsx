'use client';

import { Fragment, useState } from 'react';
import { type Route } from 'next';
import Link from 'next/link';
import { usePathname, useRouter, useSelectedLayoutSegments } from 'next/navigation';
import { Listbox, Transition } from '@headlessui/react';
import { RiCloudLine, RiExpandUpDownLine, RiLoopLeftLine, RiSettings3Line } from '@remixicon/react';

import cn from '@/utils/cn';
import { EnvironmentType, type Environment } from '@/utils/environments';

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

type Env = {
  name: string;
  slug: string;
  type: EnvironmentType;
};

type Props = {
  activeEnv: Env | undefined;
  branchEnvs: Env[];
  customEnvs: Env[];
  prodEnv: Env;
};

export default function EnvironmentSelectMenu({
  activeEnv,
  branchEnvs,
  customEnvs,
  prodEnv,
}: Props) {
  const router = useRouter();
  const [selected, setSelected] = useState<Env>();
  const nextPathname = useSwitchablePathname();

  if (!selected && activeEnv) {
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
        <div className="relative self-stretch border-x border-slate-800">
          <Listbox.Button className="font-regular flex h-full w-28 items-center gap-0.5 py-1.5 pl-4 pr-4 text-sm tracking-wide  text-white transition-all hover:bg-slate-800 lg:w-36 xl:w-[180px]">
            <span className="flex max-w-full items-center pr-4">
              {isBranchParentSelected ? (
                <>
                  <RiSettings3Line className="mr-2 h-4" />
                  <span className="block truncate">Branch Environments</span>
                </>
              ) : selected ? (
                <>
                  <span className={'mr-2 h-2 w-2 flex-shrink-0 rounded-full bg-cyan-500'} />
                  <span className="block truncate">
                    {selected.type === EnvironmentType.BranchParent
                      ? 'Branch Environments'
                      : selected.name}
                  </span>
                </>
              ) : (
                <>
                  <RiCloudLine className="mr-2 h-4 w-4" />
                  <span className="block truncate">All Environments</span>
                </>
              )}
            </span>
            <span className="pointer-events-none absolute inset-y-0 right-2 flex items-center">
              <RiExpandUpDownLine className="h-5 w-5 text-gray-400" aria-hidden="true" />
            </span>
          </Listbox.Button>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="bg-slate-940/95 absolute left-0 z-10 mt-2 w-[280px] origin-top-right divide-y divide-dashed divide-slate-700 rounded-md text-sm backdrop-blur focus:outline-none">
              <EnvironmentItem environment={prodEnv} />

              {customEnvs.map((env) => (
                <EnvironmentItem key={env.slug} environment={env} />
              ))}

              <div>
                <div className="px-4 py-3 pb-1 font-medium text-white">Branch Environments</div>
                {branchEnvs.length > 0 ? (
                  branchEnvs.map((env) => (
                    <EnvironmentItem key={env.slug} environment={env} variant="compact" />
                  ))
                ) : (
                  <Link
                    href="/env"
                    className="flex items-center gap-2.5 px-3.5 py-3 font-medium text-slate-300 transition-all hover:bg-slate-800 hover:text-white"
                  >
                    <RiLoopLeftLine className="h-3" />
                    Sync a branch
                  </Link>
                )}
              </div>

              <div className="flex items-center">
                <Link
                  prefetch={true}
                  href="/env"
                  className="flex w-full cursor-pointer items-center gap-2 truncate rounded px-3.5 py-3 text-sm text-slate-50 transition-all hover:bg-slate-700 hover:text-white"
                >
                  <RiCloudLine className="h-3 w-3" />
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
  environment: { name: string; slug: string };
  name?: string;
  variant?: 'compact' | 'normal';
}) {
  return (
    <Listbox.Option
      key={environment.slug}
      value={environment}
      className={cn(
        'flex cursor-pointer items-center gap-3 rounded px-4 py-3 text-sm font-medium text-slate-300 transition-all hover:bg-slate-800 hover:text-white',
        variant === 'compact' && 'py-2'
      )}
    >
      <span className={'block h-2 w-2 shrink-0 rounded-full bg-teal-500'} />
      <span className="truncate">{name || environment.name}</span>
    </Listbox.Option>
  );
}
