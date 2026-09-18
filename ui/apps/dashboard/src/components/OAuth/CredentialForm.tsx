import { useState, type ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Input } from '@inngest/components/Forms/Input';
import {
  Select,
  SelectWithSearch,
  type Option,
} from '@inngest/components/Select/Select';
import ToggleGroup from '@inngest/components/ToggleGroup/ToggleGroup';

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
  branchEnvironment?: Option;
  environmentGroups: EnvironmentOptionGroup[];
  onEnvironmentChange: (environment: Option | null) => void;
  permissions: ReactNode;
  selectedResourceCount: number;
  error?: ReactNode;
  actions: ReactNode;
  disabled: boolean;
};

export function CredentialForm({
  name,
  nameLabel,
  namePlaceholder,
  onNameChange,
  expiration,
  allEnvironments,
  onAllEnvironmentsChange,
  environment,
  branchEnvironment,
  environmentGroups,
  onEnvironmentChange,
  permissions,
  selectedResourceCount,
  error,
  actions,
  disabled,
}: Props) {
  const [query, setQuery] = useState('');
  const allBranches = Boolean(
    branchEnvironment && environment?.id === branchEnvironment.id,
  );
  const groups = environmentGroups
    .map((group) => ({
      ...group,
      opts: group.opts.filter((option) =>
        option.name.toLowerCase().includes(query.trim().toLowerCase()),
      ),
    }))
    .filter((group) => group.opts.length > 0);

  return (
    <div className="flex w-full flex-col gap-8">
      <fieldset disabled={disabled} className="flex flex-col gap-6">
        <legend className="sr-only">Credential details</legend>
        <div className="flex flex-col gap-2">
          <span className="text-basis text-sm font-medium">
            Choose environment
          </span>
          <div className="flex flex-wrap gap-2">
            <ToggleGroup
              type="single"
              size="small"
              aria-label="Environment access"
              value={
                allEnvironments ? 'all' : allBranches ? 'branches' : 'single'
              }
              disabled={disabled}
              onValueChange={(value) => {
                if (!value) return;
                onAllEnvironmentsChange(value === 'all');
                if (value === 'branches' && branchEnvironment) {
                  onEnvironmentChange(branchEnvironment);
                } else if (value === 'single' && allBranches) {
                  onEnvironmentChange(null);
                }
              }}
            >
              {[
                { value: 'single', label: 'Single' },
                ...(branchEnvironment
                  ? [{ value: 'branches', label: 'All branches' }]
                  : []),
                { value: 'all', label: 'All' },
              ].map(({ value, label }) => (
                <ToggleGroup.Item
                  key={value}
                  value={value}
                  className="bg-canvasSubtle hover:bg-canvasMuted hover:text-basis data-[state=on]:bg-canvasBase data-[state=on]:text-basis focus-visible:ring-primary-moderate min-w-16 px-3 focus-visible:ring-2 focus-visible:ring-inset"
                >
                  {label}
                </ToggleGroup.Item>
              ))}
            </ToggleGroup>
            {!allEnvironments && !allBranches && (
              <div className="min-w-48 flex-1">
                <SelectWithSearch
                  label="Environment"
                  isLabelVisible={false}
                  value={environment}
                  onChange={(option: Option) => {
                    onEnvironmentChange(option);
                    setQuery('');
                  }}
                >
                  <SelectWithSearch.Button>
                    <span
                      className={
                        environment ? 'text-basis truncate' : 'text-muted'
                      }
                    >
                      {environment?.name ?? 'Select an environment'}
                    </span>
                  </SelectWithSearch.Button>
                  <SelectWithSearch.Options className="w-full">
                    <SelectWithSearch.SearchInput
                      displayValue={() => query}
                      placeholder="Search environments"
                      aria-label="Search environments"
                      onChange={(event) => setQuery(event.target.value)}
                    />
                    <div className="max-h-60 overflow-y-auto">
                      {groups.map(({ label, opts }) => (
                        <div key={label}>
                          <div className="text-muted px-4 pb-1 pt-2 text-xs font-medium uppercase">
                            {label}
                          </div>
                          {opts.map((option) => (
                            <SelectWithSearch.Option
                              key={option.id}
                              option={option}
                            >
                              {option.name}
                            </SelectWithSearch.Option>
                          ))}
                        </div>
                      ))}
                      {groups.length === 0 && (
                        <p className="text-muted px-4 py-2 text-sm">
                          No environments found.
                        </p>
                      )}
                    </div>
                  </SelectWithSearch.Options>
                </SelectWithSearch>
              </div>
            )}
          </div>
          {allEnvironments && (
            <Alert severity="info">
              Environment-specific requests must specify the environment name.
              This access includes production.
            </Alert>
          )}
          {!allEnvironments && allBranches && (
            <Alert severity="info">
              Access includes current and future branch environments, not
              production. Specify a branch name for environment-specific
              requests.
            </Alert>
          )}
        </div>

        {expiration && (
          <div className="flex flex-col gap-2">
            <span className="text-basis text-sm font-medium">Expiration</span>
            <Select
              label="Expiration"
              isLabelVisible={false}
              value={expiration.value}
              onChange={expiration.onChange}
            >
              <Select.Button className="h-10">
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
        <Input
          id="credential-name"
          label={nameLabel}
          placeholder={namePlaceholder}
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          disabled={disabled}
        />
      </fieldset>

      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <span className="text-basis text-sm font-medium">Permissions</span>
          <span className="text-subtle text-xs">
            {selectedResourceCount}{' '}
            {selectedResourceCount === 1 ? 'resource' : 'resources'} selected
          </span>
        </div>
        {permissions}
      </div>
      {error}
      <div className="flex gap-2">{actions}</div>
    </div>
  );
}
