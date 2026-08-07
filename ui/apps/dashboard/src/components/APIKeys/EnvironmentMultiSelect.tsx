import { useEffect, useMemo, useRef, useState } from 'react';
import {
  SelectWithSearch,
  type Option,
} from '@inngest/components/Select/Select';
import { cn } from '@inngest/components/utils/classNames';
import { RiCloseLine, RiLockLine } from '@remixicon/react';

import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';

type EnvGroups = {
  production: Option[];
  test: Option[];
  branches: Option[];
};

/**
 * `active` gates the Production auto-select, e.g. only while a modal is open.
 *
 * `allowProduction` false drops Production from the list rather than rendering
 * it disabled, because CheckboxOption doesn't forward `disabled` to the option
 * and a "disabled" row would still be selectable.
 */
export function useEnvironmentSelection({
  active = true,
  allowProduction = true,
}: { active?: boolean; allowProduction?: boolean } = {}) {
  const [selectedEnvs, setSelectedEnvs] = useState<Option[]>([]);
  const [{ data: envs }] = useEnvironments();

  // Branch-env keys live on the parent, which authenticates every child, so
  // offer the parent and hide the children.
  const envGroups = useMemo<EnvGroups>(() => {
    const production: Option[] = [];
    const test: Option[] = [];
    const branches: Option[] = [];
    for (const e of envs ?? []) {
      if (e.isArchived || e.type === EnvironmentType.BranchChild) continue;
      const opt = { id: e.id, name: e.name };
      if (e.type === EnvironmentType.Production) {
        if (allowProduction) production.push(opt);
      } else if (e.type === EnvironmentType.BranchParent) {
        branches.push(opt);
      } else {
        test.push(opt);
      }
    }
    return { production, test, branches };
  }, [envs, allowProduction]);

  // Auto-select Production only when there is exactly one; with several, the
  // user should choose.
  //
  // Runs once per activation rather than whenever the selection is empty:
  // keying off empty makes Select none unobservable, since clearing re-runs the
  // effect and re-adds Production. Resetting on `active` false means a reopened
  // modal still defaults to Production.
  const autoSelected = useRef(false);
  useEffect(() => {
    if (!active) {
      autoSelected.current = false;
      return;
    }
    if (autoSelected.current || selectedEnvs.length > 0) return;
    const only = envGroups.production[0];
    if (envGroups.production.length === 1 && only) {
      autoSelected.current = true;
      setSelectedEnvs([only]);
    }
  }, [active, selectedEnvs, envGroups.production]);

  return { selectedEnvs, setSelectedEnvs, envGroups };
}

type Props = {
  groups: EnvGroups;
  value: Option[];
  onChange: (envs: Option[]) => void;
  /** Explains the absent Production row when the account policy withholds it. */
  productionNote?: string;
  disabled?: boolean;
};

const GROUP_LABELS: [keyof EnvGroups, string][] = [
  ['production', 'Production'],
  ['test', 'Test'],
  ['branches', 'Branch environments'],
];

export function EnvironmentMultiSelect({
  groups,
  value,
  onChange,
  productionNote,
  disabled = false,
}: Props) {
  const [search, setSearch] = useState('');

  const all = useMemo(
    () => [...groups.production, ...groups.test, ...groups.branches],
    [groups],
  );
  const productionIDs = useMemo(
    () => new Set(groups.production.map((o) => o.id)),
    [groups.production],
  );
  const term = search.trim().toLowerCase();
  const matches = (opt: Option) =>
    term === '' || opt.name.toLowerCase().includes(term);

  function remove(id: string) {
    onChange(value.filter((o) => o.id !== id));
  }

  return (
    <div className="flex flex-col gap-1.5">
      {/* Above the control rather than inside the dropdown: the options panel is
          a listbox, and browsers prune non-options from its accessibility tree,
          so bulk actions placed in there are invisible to a screen reader. */}
      <div className="flex items-center justify-between gap-3">
        <span className="text-light text-xs">
          {value.length} of {all.length} selected
        </span>
        <span className="flex gap-3">
          <button
            type="button"
            className="text-link text-xs"
            disabled={disabled}
            onClick={() => onChange(all)}
          >
            Select all
          </button>
          <button
            type="button"
            className="text-link text-xs"
            disabled={disabled}
            onClick={() => onChange([])}
          >
            Select none
          </button>
        </span>
      </div>

      <SelectWithSearch
        multiple
        label="Environments"
        isLabelVisible={false}
        value={value}
        onChange={onChange}
        className="w-full"
      >
        <SelectWithSearch.Button
          isLabelVisible={false}
          className="h-auto min-h-[32px] flex-wrap gap-1.5 py-1"
        >
          {value.length === 0 ? (
            <span className="text-disabled text-sm">
              Select one or more environments
            </span>
          ) : (
            <span className="flex flex-wrap gap-1.5">
              {value.map((env) => (
                <span
                  key={env.id}
                  className="border-subtle bg-canvasSubtle text-basis flex h-[22px] items-center gap-1.5 rounded-2xl border pl-2 pr-1.5 text-xs font-medium"
                >
                  {env.name}
                  {productionIDs.has(env.id) && (
                    <span className="bg-success h-1.5 w-1.5 shrink-0 rounded-full" />
                  )}
                  {/* A <button> here would nest inside the combobox trigger,
                      which is itself a button. */}
                  <span
                    role="button"
                    tabIndex={disabled ? -1 : 0}
                    aria-label={`Remove ${env.name}`}
                    className="text-muted hover:text-basis"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      if (!disabled) remove(env.id);
                    }}
                    onKeyDown={(e) => {
                      if (e.key !== 'Enter' && e.key !== ' ') return;
                      e.preventDefault();
                      e.stopPropagation();
                      if (!disabled) remove(env.id);
                    }}
                  >
                    <RiCloseLine className="h-3 w-3" />
                  </span>
                </span>
              ))}
            </span>
          )}
        </SelectWithSearch.Button>

        <SelectWithSearch.Options className="w-full">
          <SelectWithSearch.SearchInput
            displayValue={() => search}
            placeholder="Search environments"
            onChange={(e) => setSearch(e.target.value)}
          />
          <div className="max-h-60 overflow-auto">
            {GROUP_LABELS.map(([key, label]) => {
              const opts = groups[key].filter(matches);
              if (opts.length === 0) return null;
              return (
                <div key={key}>
                  <div className="text-light px-4 pb-1 pt-1.5 text-xs font-medium uppercase tracking-wide">
                    {label}
                  </div>
                  {opts.map((opt) => (
                    <SelectWithSearch.CheckboxOption key={opt.id} option={opt}>
                      <span
                        className={cn(
                          'text-sm',
                          key === 'production' ? '' : 'font-mono text-xs',
                        )}
                      >
                        {opt.name}
                      </span>
                    </SelectWithSearch.CheckboxOption>
                  ))}
                </div>
              );
            })}
            {all.filter(matches).length === 0 && (
              <p className="text-light px-4 py-2 text-sm">
                No environments match “{search.trim()}”.
              </p>
            )}
          </div>
        </SelectWithSearch.Options>
      </SelectWithSearch>

      {productionNote && (
        <div className="text-light flex items-center gap-1.5 text-xs">
          <RiLockLine className="h-3 w-3 shrink-0" />
          {productionNote}
        </div>
      )}
    </div>
  );
}
