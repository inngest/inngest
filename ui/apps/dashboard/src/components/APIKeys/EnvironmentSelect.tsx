import { useEffect, useMemo, useState } from 'react';
import { Select, type Option } from '@inngest/components/Select/Select';

import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';

type EnvGroups = {
  production: Option[];
  test: Option[];
  branches: Option[];
};

// Environment choice for minting an API key, shared by the create-key modal
// and the device-login approval page so both offer the same environments.
//
// `active` gates the Production auto-select (e.g. only while a modal is open).
export function useEnvironmentSelection(active = true) {
  const [selectedEnv, setSelectedEnv] = useState<Option | null>(null);
  const [{ data: envs }] = useEnvironments();

  // Pickable envs split by type so the picker can render Production / Test /
  // Branches groups instead of one alphabetical blob. Keys for branch envs
  // live on the parent (mirrors how signing and event keys work) — a
  // parent-scoped key authenticates for every child beneath it, so we offer
  // the parent and hide the programmatically-created children.
  const envGroups = useMemo<EnvGroups>(() => {
    const production: Option[] = [];
    const test: Option[] = [];
    const branches: Option[] = [];
    for (const e of envs ?? []) {
      if (e.isArchived || e.type === EnvironmentType.BranchChild) continue;
      const opt = { id: e.id, name: e.name };
      if (e.type === EnvironmentType.Production) production.push(opt);
      else if (e.type === EnvironmentType.BranchParent) branches.push(opt);
      else test.push(opt);
    }
    return { production, test, branches };
  }, [envs]);

  // Pre-select Production so the common case is one click. Only auto-select
  // when there's exactly one production env — a user with several should make
  // an explicit choice.
  useEffect(() => {
    if (!active || selectedEnv) return;
    if (envGroups.production.length === 1) {
      setSelectedEnv(envGroups.production[0] ?? null);
    }
  }, [active, selectedEnv, envGroups.production]);

  return { selectedEnv, setSelectedEnv, envGroups };
}

type Props = {
  groups: EnvGroups;
  value: Option | null;
  onChange: (opt: Option) => void;
};

export function EnvironmentSelect({ groups, value, onChange }: Props) {
  return (
    <Select
      label="Environment"
      isLabelVisible={false}
      value={value}
      onChange={onChange}
    >
      <Select.Button>
        <span className={value ? 'text-basis' : 'text-disabled'}>
          {value?.name ?? 'Select an environment'}
        </span>
      </Select.Button>
      <Select.Options>
        {(
          [
            ['Production', groups.production],
            ['Test', groups.test],
            ['Branches', groups.branches],
          ] as const
        ).map(([label, opts], idx) =>
          opts.length === 0 ? null : (
            <div key={label}>
              {idx > 0 && <hr className="border-subtle my-1" />}
              <div className="text-light px-4 pb-1 pt-1.5 text-xs font-medium uppercase tracking-wide">
                {label}
              </div>
              {opts.map((opt) => (
                <Select.Option key={opt.id} option={opt}>
                  {opt.name}
                </Select.Option>
              ))}
            </div>
          ),
        )}
      </Select.Options>
    </Select>
  );
}
