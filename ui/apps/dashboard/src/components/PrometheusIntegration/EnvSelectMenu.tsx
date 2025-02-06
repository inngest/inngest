import { useState } from 'react';
import { Listbox } from '@headlessui/react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiCloudLine, RiErrorWarningLine, RiExpandUpDownLine } from '@remixicon/react';

import { useEnvironments } from '@/queries';
import { getDefaultEnvironment, getTestEnvironments, type Environment } from '@/utils/environments';

type EnvSelectMenuProps = {
  onSelect?: (env: Environment) => void;
};

// EnvSelectMenu is a dropdown menu that allows selecting a single environment.
// It does not allow selecting branch or archived envs.
// It defaults to the Production env.
export default function EnvSelectMenu({ onSelect }: EnvSelectMenuProps) {
  const [{ data: envs = [], error }] = useEnvironments();
  const [selection, setSelection] = useState<Environment | null>(null);

  if (error) {
    console.error('error fetching envs', error);
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="bg-error text-error flex h-8 w-full items-center justify-start gap-x-2 rounded px-2">
            <RiErrorWarningLine className="w-4" />
            <div>Error</div>
          </div>
        </TooltipTrigger>
        <TooltipContent side="right" className="text-error bg-error rounded text-xs">
          Error loading environments. Please refresh the page, and contact support if this keeps
          happening.
        </TooltipContent>
      </Tooltip>
    );
  }

  const defaultEnvironment = getDefaultEnvironment(envs);
  const includeArchived = false;
  const testEnvironments = getTestEnvironments(envs, includeArchived);

  const internalOnSelect = (env: Environment) => {
    if (onSelect) {
      onSelect(env);
    }
    setSelection(env);
  };

  if (selection === null && defaultEnvironment) {
    internalOnSelect(defaultEnvironment);
  }

  return (
    <Listbox value={selection} onChange={internalOnSelect}>
      {({ open }) => (
        <div className="bg-canvasBase relative flex">
          <Listbox.Button
            className={`border-muted ${open && 'border-primary-intense'}
            bg-canvasBase text-primary-intense hover:bg-canvasSubtle 
            h-8 w-[258px] overflow-hidden rounded border px-2 text-sm`}
          >
            <div className="flex flex-row items-center justify-between">
              <SelectedDisplay env={selection} />
              <RiExpandUpDownLine className="text-muted h-4 w-4" aria-hidden="true" />
            </div>
          </Listbox.Button>

          <Listbox.Options
            className="bg-canvasBase border-subtle overflow-y-truncate absolute
            left-0.5 top-9 z-50 w-[250px] divide-none rounded border shadow focus:outline-none"
          >
            {defaultEnvironment !== null && (
              <EnvItem env={defaultEnvironment} key={defaultEnvironment.id} />
            )}
            {testEnvironments.length > 0 &&
              testEnvironments.map((env) => <EnvItem key={env.id} env={env} />)}
          </Listbox.Options>
        </div>
      )}
    </Listbox>
  );
}

const SelectedDisplay = ({ env }: { env: Environment | null }) => (
  <span className="flex min-w-0 flex-row items-center truncate">
    {env ? (
      <span className="block">{env.name}</span>
    ) : (
      <>
        <RiCloudLine className="mr-2 h-4 w-4" />
        <span className="block">Loading...</span>
      </>
    )}
  </span>
);

const EnvItem = ({ env }: { env: Environment }) => (
  <Listbox.Option
    key={'lo-' + env.id}
    value={env}
    className="bg-canvasBase hover:bg-canvasSubtle text-subtle flex h-10 cursor-pointer items-center gap-3 px-3 text-[13px] font-normal"
  >
    <span className="bg-primary-moderate block h-1.5 w-1.5 shrink-0 rounded-full" />
    <span className="truncate">{env.name}</span>
  </Listbox.Option>
);
