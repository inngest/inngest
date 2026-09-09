import type { ReactNode } from 'react';
import { Input } from '@inngest/components/Forms/Input';
import { Select, type Option } from '@inngest/components/Select/Select';

export type EnvironmentOptionGroup = { label: string; opts: Option[] };

type Props = {
  name: string;
  nameLabel: string;
  namePlaceholder?: string;
  onNameChange: (name: string) => void;
  expiration?: {
    value: Option;
    options: Option[];
    onChange: (option: Option) => void;
  };
  allEnvironments: boolean;
  onAllEnvironmentsChange: (all: boolean) => void;
  environment: Option | null;
  environmentGroups: EnvironmentOptionGroup[];
  onEnvironmentChange: (environment: Option) => void;
  permissions: ReactNode;
  selectedResourceCount: number;
  error?: ReactNode;
  actions: ReactNode;
  disabled: boolean;
};

// Presentation for OAuth consent. The page owns the catalog, defaults,
// validation, and submission flow.
export function CredentialForm({
  name,
  nameLabel,
  namePlaceholder,
  onNameChange,
  expiration,
  allEnvironments,
  onAllEnvironmentsChange,
  environment,
  environmentGroups,
  onEnvironmentChange,
  permissions,
  selectedResourceCount,
  error,
  actions,
  disabled,
}: Props) {
  const boundaryName = allEnvironments
    ? 'All environments'
    : 'Single environment';
  const groups = environmentGroups.filter((group) => group.opts.length > 0);

  return (
    <div className="flex w-full flex-col gap-6">
      <fieldset
        disabled={disabled}
        className="grid grid-cols-1 gap-5 sm:grid-cols-2"
      >
        <legend className="sr-only">Credential details</legend>
        <div className={expiration ? '' : 'sm:col-span-2'}>
          <Input
            id="credential-name"
            label={nameLabel}
            placeholder={namePlaceholder}
            value={name}
            onChange={(event) => onNameChange(event.target.value)}
            disabled={disabled}
          />
        </div>
        {expiration && (
          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">Expiration</label>
            <Select
              className="h-8"
              label="Expiration"
              isLabelVisible={false}
              value={expiration.value}
              onChange={expiration.onChange}
            >
              <Select.Button className="h-[30px] py-0">
                <span className="text-basis">{expiration.value.name}</span>
              </Select.Button>
              <Select.Options>
                {expiration.options.map((option) => (
                  <Select.Option key={option.id} option={option}>
                    {option.name}
                  </Select.Option>
                ))}
              </Select.Options>
            </Select>
          </div>
        )}

        <div className="flex flex-col gap-3 sm:col-span-2">
          <label className="text-basis text-sm font-medium">Boundary</label>
          <div className="border-subtle grid grid-cols-1 gap-5 rounded border p-3 sm:grid-cols-2">
            <div className="flex flex-col gap-1">
              <label className="text-basis text-sm font-medium">Mode</label>
              <Select
                className="h-8"
                label="Boundary"
                isLabelVisible={false}
                value={{
                  id: allEnvironments ? 'all' : 'single',
                  name: boundaryName,
                }}
                onChange={(option) =>
                  onAllEnvironmentsChange(option.id === 'all')
                }
              >
                <Select.Button className="h-[30px] py-0">
                  <span className="text-basis">{boundaryName}</span>
                </Select.Button>
                <Select.Options>
                  <Select.Option
                    option={{ id: 'single', name: 'Single environment' }}
                  >
                    Single environment
                  </Select.Option>
                  <Select.Option
                    option={{ id: 'all', name: 'All environments' }}
                  >
                    All environments
                  </Select.Option>
                </Select.Options>
              </Select>
            </div>

            {!allEnvironments && (
              <div className="flex flex-col gap-1">
                <label className="text-basis text-sm font-medium">
                  Environment
                </label>
                <Select
                  className="h-8"
                  label="Environment"
                  isLabelVisible={false}
                  value={environment}
                  onChange={onEnvironmentChange}
                >
                  <Select.Button className="h-[30px] py-0">
                    <span
                      className={environment ? 'text-basis' : 'text-disabled'}
                    >
                      {environment?.name ?? 'Select an environment'}
                    </span>
                  </Select.Button>
                  <Select.Options>
                    {groups.map(({ label, opts }, index) => (
                      <div key={label}>
                        {index > 0 && <hr className="border-subtle my-1" />}
                        <div className="text-light px-4 pb-1 pt-1.5 text-xs font-medium uppercase tracking-wide">
                          {label}
                        </div>
                        {opts.map((option) => (
                          <Select.Option key={option.id} option={option}>
                            {option.name}
                          </Select.Option>
                        ))}
                      </div>
                    ))}
                  </Select.Options>
                </Select>
              </div>
            )}
            {allEnvironments && (
              <div className="bg-canvasSubtle border-subtle text-subtle rounded border px-3 py-2 text-sm sm:col-span-2">
                Environment-specific requests must specify the environment name.
                This access includes production.
              </div>
            )}
          </div>
        </div>
      </fieldset>

      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <label className="text-basis text-sm font-medium">Permissions</label>
          <span className="text-subtle text-xs">
            {selectedResourceCount} resources selected
          </span>
        </div>
        {permissions}
      </div>
      {error}
      <div className="flex justify-end gap-2">{actions}</div>
    </div>
  );
}
