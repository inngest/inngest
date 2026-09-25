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
  nameRequired?: boolean;
  nameFirst?: boolean;
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
  fieldErrors?: {
    name?: string | null;
    environment?: string | null;
    permissions?: string | null;
  };
  actions: ReactNode;
  disabled: boolean;
};

export function CredentialForm({
  name,
  nameLabel,
  namePlaceholder,
  nameRequired = false,
  nameFirst = false,
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
  fieldErrors,
  actions,
  disabled,
}: Props) {
  const [query, setQuery] = useState('');
  const groups = environmentGroups
    .map((group) => ({
      ...group,
      opts: group.opts.filter((option) =>
        option.name.toLowerCase().includes(query.trim().toLowerCase()),
      ),
    }))
    .filter((group) => group.opts.length > 0);

  const nameInput = (
    <Input
      id="credential-name"
      name="credential-name"
      label={nameLabel}
      required={nameRequired}
      error={fieldErrors?.name ?? undefined}
      aria-invalid={Boolean(fieldErrors?.name)}
      placeholder={namePlaceholder}
      value={name}
      onChange={(event) => onNameChange(event.target.value)}
      disabled={disabled}
    />
  );

  return (
    <div className="flex w-full flex-col gap-8">
      <fieldset disabled={disabled} className="flex flex-col gap-6">
        <legend className="sr-only">Credential details</legend>
        {nameFirst && nameInput}
        <div
          className="flex flex-col gap-2"
          role="group"
          aria-label="Environment"
          aria-invalid={Boolean(fieldErrors?.environment)}
          aria-describedby={
            fieldErrors?.environment
              ? 'credential-environment-error'
              : undefined
          }
          tabIndex={-1}
        >
          <span className="text-basis text-sm font-medium">
            Choose environment{' '}
            <span className="text-subtle font-normal">(required)</span>
          </span>
          <div className="flex flex-wrap gap-2">
            <ToggleGroup
              type="single"
              size="small"
              aria-label="Environment access"
              value={allEnvironments ? 'all' : 'single'}
              disabled={disabled}
              onValueChange={(value) => {
                if (value) {
                  onAllEnvironmentsChange(value === 'all');
                }
              }}
            >
              {[
                { value: 'single', label: 'Single' },
                { value: 'all', label: 'All' },
              ].map(({ value, label }) => (
                <ToggleGroup.Item
                  key={value}
                  value={value}
                  className="bg-canvasSubtle hover:bg-canvasMuted hover:text-basis data-[state=on]:bg-canvasBase data-[state=on]:text-basis focus-visible:ring-primary-moderate px-3 focus-visible:ring-2 focus-visible:ring-inset"
                >
                  {label}
                </ToggleGroup.Item>
              ))}
            </ToggleGroup>
            {!allEnvironments && (
              <div className="min-w-48 flex-1">
                <SelectWithSearch
                  label="Environment"
                  className={
                    fieldErrors?.environment ? 'border-error' : undefined
                  }
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
          {fieldErrors?.environment && (
            <p id="credential-environment-error" className="text-error text-sm">
              {fieldErrors.environment}
            </p>
          )}
          {allEnvironments && (
            <Alert severity="info">
              Environment-specific requests must specify the environment name.
              This access includes production.
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
        {!nameFirst && nameInput}
      </fieldset>

      <div
        className="flex flex-col gap-3"
        role="group"
        aria-label="Permissions"
        aria-invalid={Boolean(fieldErrors?.permissions)}
        aria-describedby={
          fieldErrors?.permissions ? 'credential-permissions-error' : undefined
        }
        tabIndex={-1}
      >
        <div className="flex items-center justify-between gap-3">
          <span className="text-basis text-sm font-medium">
            Permissions{' '}
            <span className="text-subtle font-normal">(required)</span>
          </span>
          <span className="text-subtle text-xs">
            {selectedResourceCount}{' '}
            {selectedResourceCount === 1 ? 'resource' : 'resources'} selected
          </span>
        </div>
        {permissions}
        {fieldErrors?.permissions && (
          <p id="credential-permissions-error" className="text-error text-sm">
            {fieldErrors.permissions}
          </p>
        )}
      </div>
      {error}
      <div className="flex gap-2">{actions}</div>
    </div>
  );
}
