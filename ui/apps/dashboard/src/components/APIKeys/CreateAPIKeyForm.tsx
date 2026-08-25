import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Select, type Option } from '@inngest/components/Select/Select';
import { gql, useMutation, useQuery, type TypedDocumentNode } from 'urql';

import { graphql } from '@/gql';
import {
  ApiKeyCredentialSource,
  ApiKeyOwnershipType,
  ApiKeyResourceBoundaryMode,
  type CreateApiKeyInput,
} from '@/gql/graphql';
import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';
import { apiKeyErrorMessage } from './errorMessage';
import {
  PermissionPicker,
  type PermissionGroup,
  type PermissionLevel,
} from './PermissionPicker';
import { RevealKeyCard } from './RevealKeyCard';
import { validateAPIKeyName } from './validation';

const Mutation = graphql(`
  mutation CreateAPIKey($input: CreateAPIKeyInput!) {
    createAPIKey(input: $input) {
      plaintextKey
      apiKey {
        id
        name
        createdAt
        maskedKey
        env {
          id
          name
        }
      }
    }
  }
`);

type PermissionCatalogResult = {
  apiKeyPermissionCatalog: PermissionGroup[];
};

type EnvOptionGroup = {
  label: string;
  opts: Option[];
};
type BoundaryOptionID =
  | 'single_environment'
  | 'all_branch_environments'
  | 'all_environments';
type ExpirationOptionID =
  | 'seven_days'
  | 'thirty_days'
  | 'ninety_days'
  | 'one_year'
  | 'never';
type ExpirationOption = {
  id: ExpirationOptionID;
  label: string;
  days: number | null;
  serviceOnly: boolean;
};

const EXPIRATION_OPTIONS: ExpirationOption[] = [
  {
    id: 'seven_days',
    label: '7 days',
    days: 7,
    serviceOnly: false,
  },
  {
    id: 'thirty_days',
    label: '30 days',
    days: 30,
    serviceOnly: false,
  },
  {
    id: 'ninety_days',
    label: '90 days',
    days: 90,
    serviceOnly: false,
  },
  {
    id: 'one_year',
    label: '1 year',
    days: 365,
    serviceOnly: false,
  },
  {
    id: 'never',
    label: 'No expiration',
    days: null,
    serviceOnly: true,
  },
];

const PermissionCatalogQuery: TypedDocumentNode<
  PermissionCatalogResult,
  Record<string, never>
> = gql`
  query APIKeyPermissionCatalog {
    apiKeyPermissionCatalog {
      resource
      read
      write
    }
  }
`;

function boundaryOptionName(optionID: BoundaryOptionID) {
  switch (optionID) {
    case 'single_environment':
      return 'Single environment';
    case 'all_branch_environments':
      return 'All branch environments';
    case 'all_environments':
      return 'All environments';
  }
}

function boundaryOptionIDFromOption(opt: Option): BoundaryOptionID {
  switch (opt.id) {
    case 'all_branch_environments':
      return 'all_branch_environments';
    case 'all_environments':
      return 'all_environments';
  }
  return 'single_environment';
}

function getDefaultOwnershipType(createUserAPIKey: boolean) {
  if (createUserAPIKey) {
    return ApiKeyOwnershipType.User;
  }
  return ApiKeyOwnershipType.Service;
}

function getDefaultExpirationOption(ownershipType: ApiKeyOwnershipType) {
  if (ownershipType === ApiKeyOwnershipType.User) {
    return 'thirty_days';
  }
  return 'one_year';
}

function expirationOptionName(optionID: ExpirationOptionID) {
  for (const option of EXPIRATION_OPTIONS) {
    if (option.id === optionID) {
      return option.label;
    }
  }
  return '30 days';
}

function expirationOptionIDFromOption(opt: Option): ExpirationOptionID {
  switch (opt.id) {
    case 'seven_days':
      return 'seven_days';
    case 'thirty_days':
      return 'thirty_days';
    case 'ninety_days':
      return 'ninety_days';
    case 'one_year':
      return 'one_year';
    case 'never':
      return 'never';
  }
  return 'thirty_days';
}

function expiresAtForOption(optionID: ExpirationOptionID) {
  for (const option of EXPIRATION_OPTIONS) {
    if (option.id !== optionID) continue;
    if (option.days === null) return null;

    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + option.days);
    return expiresAt.toISOString();
  }
  return null;
}

type Props = {
  createUserAPIKey: boolean;
  canCreateServiceKeys: boolean;
  onCancel: () => void;
  onDone: () => void;
};

export function CreateAPIKeyForm({
  createUserAPIKey,
  canCreateServiceKeys,
  onCancel,
  onDone,
}: Props) {
  const defaultOwnershipType = getDefaultOwnershipType(createUserAPIKey);

  const [name, setName] = useState('');
  const [selectedEnv, setSelectedEnv] = useState<Option | null>(null);
  const [boundaryOption, setBoundaryOption] =
    useState<BoundaryOptionID>('single_environment');
  const [ownershipType, setOwnershipType] =
    useState<ApiKeyOwnershipType>(defaultOwnershipType);
  const [expirationOption, setExpirationOption] = useState<ExpirationOptionID>(
    getDefaultExpirationOption(defaultOwnershipType),
  );
  const [permissionLevels, setPermissionLevels] = useState<
    Record<string, PermissionLevel>
  >({});
  const [plaintextKey, setPlaintextKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const cancelledRef = useRef(false);
  const catalogDefaultsAppliedRef = useRef(false);

  const [{ data: envs }] = useEnvironments();
  const [, create] = useMutation(Mutation);
  const [catalogRes] = useQuery({
    query: PermissionCatalogQuery,
    requestPolicy: 'cache-first',
    variables: {},
  });

  const envGroups = useMemo(() => {
    const production: Option[] = [];
    const test: Option[] = [];
    let branchParent: Option | null = null;
    for (const e of envs ?? []) {
      if (e.isArchived || e.type === EnvironmentType.BranchChild) continue;
      const opt = { id: e.id, name: e.name };
      if (e.type === EnvironmentType.Production) production.push(opt);
      else if (e.type === EnvironmentType.BranchParent) branchParent = opt;
      else test.push(opt);
    }
    return { production, test, branchParent };
  }, [envs]);

  const envOptionGroups: EnvOptionGroup[] = [
    { label: 'Production', opts: envGroups.production },
    { label: 'Test', opts: envGroups.test },
  ];

  const selectedPermissions = useMemo(() => {
    const grants = new Set<string>();
    for (const group of catalogRes.data?.apiKeyPermissionCatalog ?? []) {
      const level = permissionLevels[group.resource] ?? 'none';
      if (level === 'read' || level === 'write') {
        for (const grant of group.read) {
          grants.add(grant);
        }
      }
      if (level === 'write') {
        for (const grant of group.write) {
          grants.add(grant);
        }
      }
    }
    return Array.from(grants).sort();
  }, [catalogRes.data?.apiKeyPermissionCatalog, permissionLevels]);

  const selectedResourceCount = Object.values(permissionLevels).filter(
    (level) => level !== 'none',
  ).length;

  useEffect(() => {
    const nextOwnershipType = getDefaultOwnershipType(createUserAPIKey);
    setOwnershipType(nextOwnershipType);
    setExpirationOption(getDefaultExpirationOption(nextOwnershipType));
  }, [createUserAPIKey]);

  useEffect(() => {
    if (selectedEnv) return;
    if (envGroups.production.length === 1) {
      setSelectedEnv(envGroups.production[0] ?? null);
    }
  }, [selectedEnv, envGroups.production]);

  useEffect(() => {
    if (catalogDefaultsAppliedRef.current || !catalogRes.data) return;

    const defaults: Record<string, PermissionLevel> = {};
    for (const group of catalogRes.data.apiKeyPermissionCatalog) {
      if (group.read.length > 0) {
        defaults[group.resource] = 'read';
      }
    }
    setPermissionLevels(defaults);
    catalogDefaultsAppliedRef.current = true;
  }, [catalogRes.data]);

  function setPermissionLevel(resource: string, level: PermissionLevel) {
    setPermissionLevels((prev) => ({ ...prev, [resource]: level }));
  }

  function reset() {
    setName('');
    setSelectedEnv(null);
    const nextOwnershipType = getDefaultOwnershipType(createUserAPIKey);
    setOwnershipType(nextOwnershipType);
    setBoundaryOption('single_environment');
    setExpirationOption(getDefaultExpirationOption(nextOwnershipType));
    setPermissionLevels({});
    catalogDefaultsAppliedRef.current = false;
    setPlaintextKey(null);
    setError(null);
    setIsSubmitting(false);
  }

  function cancel() {
    cancelledRef.current = true;
    reset();
    onCancel();
  }

  function done() {
    cancelledRef.current = true;
    reset();
    onDone();
  }

  async function submit() {
    setError(null);

    const nameErr = validateAPIKeyName(name);
    if (nameErr) {
      setError(nameErr);
      return;
    }

    if (catalogRes.fetching && !catalogRes.data) {
      setError('Wait for permissions to load.');
      return;
    }
    if (catalogRes.error) {
      setError(
        apiKeyErrorMessage(catalogRes.error, 'Could not load permissions.'),
      );
      return;
    }
    if (
      ownershipType === ApiKeyOwnershipType.Service &&
      !canCreateServiceKeys
    ) {
      setError('You do not have permission to create API keys.');
      return;
    }
    if (ownershipType === ApiKeyOwnershipType.User && !createUserAPIKey) {
      setError('API keys cannot be created from this page.');
      return;
    }
    if (boundaryOption === 'single_environment' && !selectedEnv) {
      setError('Select an environment.');
      return;
    }
    if (
      boundaryOption === 'all_branch_environments' &&
      !envGroups.branchParent
    ) {
      setError('Branch environments are not available for this account.');
      return;
    }
    if (selectedPermissions.length === 0) {
      setError('Select at least one permission.');
      return;
    }
    if (
      ownershipType === ApiKeyOwnershipType.User &&
      expirationOption === 'never'
    ) {
      setError('API keys must expire.');
      return;
    }

    const expiresAt = expiresAtForOption(expirationOption);
    let resourceBoundaryMode = ApiKeyResourceBoundaryMode.SingleEnv;
    let workspaceID = selectedEnv?.id ?? null;
    if (boundaryOption === 'all_environments') {
      resourceBoundaryMode = ApiKeyResourceBoundaryMode.AllEnvs;
      workspaceID = null;
    }
    if (boundaryOption === 'all_branch_environments') {
      workspaceID = envGroups.branchParent?.id ?? null;
    }

    const input: CreateApiKeyInput = {
      credentialSource: ApiKeyCredentialSource.DashboardUi,
      name: name.trim(),
      ownershipType,
      permissions: selectedPermissions,
      resourceBoundaryMode,
      workspaceID,
    };
    if (expiresAt) {
      input.expiresAt = expiresAt;
    }

    cancelledRef.current = false;
    setIsSubmitting(true);
    try {
      const res = await create({ input }, { additionalTypenames: ['APIKey'] });
      if (cancelledRef.current) return;
      if (res.error) {
        setError(apiKeyErrorMessage(res.error, 'Could not create API key.'));
        return;
      }

      const pt = res.data?.createAPIKey?.plaintextKey;
      if (!pt) {
        setError('Unexpected response from server.');
        return;
      }
      setPlaintextKey(pt);
    } finally {
      if (!cancelledRef.current) {
        setIsSubmitting(false);
      }
    }
  }

  if (plaintextKey !== null) {
    return (
      <div className="flex w-full flex-col gap-5">
        <div className="flex flex-col gap-1">
          <h2 className="text-basis text-xl">Copy your API key</h2>
        </div>
        <RevealKeyCard plaintextKey={plaintextKey} />
        <div className="flex justify-end">
          <Button kind="primary" label="Done" onClick={done} />
        </div>
      </div>
    );
  }

  let nameInputClassName = '';
  if (!createUserAPIKey) {
    nameInputClassName = 'sm:col-span-2';
  }

  let expirationField = null;
  if (createUserAPIKey) {
    const expirationOptions: ExpirationOption[] = [];
    for (const option of EXPIRATION_OPTIONS) {
      if (option.serviceOnly) {
        continue;
      }
      expirationOptions.push(option);
    }

    expirationField = (
      <div className="flex flex-col gap-1">
        <label className="text-basis text-sm font-medium">Expiration</label>
        <Select
          className="h-8"
          label="Expiration"
          isLabelVisible={false}
          value={{
            id: expirationOption,
            name: expirationOptionName(expirationOption),
          }}
          onChange={(opt) =>
            setExpirationOption(expirationOptionIDFromOption(opt))
          }
        >
          <Select.Button className="h-[30px] py-0">
            <span className="text-basis">
              {expirationOptionName(expirationOption)}
            </span>
          </Select.Button>
          <Select.Options>
            {expirationOptions.map((option) => (
              <Select.Option
                key={option.id}
                option={{
                  id: option.id,
                  name: option.label,
                }}
              >
                {option.label}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
      </div>
    );
  }

  let environmentField = null;
  let boundaryCallout = null;
  if (boundaryOption === 'single_environment') {
    let environmentLabelClassName = 'text-disabled';
    if (selectedEnv) {
      environmentLabelClassName = 'text-basis';
    }

    environmentField = (
      <div className="flex flex-col gap-1">
        <label className="text-basis text-sm font-medium">Environment</label>
        <Select
          className="h-8"
          label="Environment"
          isLabelVisible={false}
          value={selectedEnv}
          onChange={(opt) => setSelectedEnv(opt)}
        >
          <Select.Button className="h-[30px] py-0">
            <span className={environmentLabelClassName}>
              {selectedEnv?.name ?? 'Select an environment'}
            </span>
          </Select.Button>
          <Select.Options>
            {envOptionGroups.map(({ label, opts }, idx) => {
              if (opts.length === 0) {
                return null;
              }
              return (
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
              );
            })}
          </Select.Options>
        </Select>
      </div>
    );
  }
  if (boundaryOption === 'all_branch_environments') {
    boundaryCallout = (
      <div className="bg-canvasSubtle border-subtle text-subtle rounded border px-3 py-2 text-sm sm:col-span-2">
        This key applies to every current and future branch environment. API
        requests made with this key must specify the branch environment name.
      </div>
    );
  }
  if (boundaryOption === 'all_environments') {
    boundaryCallout = (
      <div className="bg-canvasSubtle border-subtle text-subtle rounded border px-3 py-2 text-sm sm:col-span-2">
        API requests made with this key must specify the environment name.
      </div>
    );
  }

  const permissionGroups = catalogRes.data?.apiKeyPermissionCatalog ?? [];
  let permissionsContent = (
    <PermissionPicker
      groups={permissionGroups}
      levels={permissionLevels}
      disabled={isSubmitting}
      onChange={setPermissionLevel}
    />
  );
  if (catalogRes.fetching && !catalogRes.data) {
    permissionsContent = (
      <div className="border-subtle text-subtle rounded border px-3 py-2 text-sm">
        Loading permissions...
      </div>
    );
  } else if (catalogRes.error) {
    permissionsContent = (
      <Alert severity="error">
        {apiKeyErrorMessage(catalogRes.error, 'Could not load permissions.')}
      </Alert>
    );
  }

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div className={nameInputClassName}>
          <Input
            id="api-key-name"
            label="API key name"
            placeholder="eg. my-api-key"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isSubmitting}
          />
        </div>
        {expirationField}

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
                  id: boundaryOption,
                  name: boundaryOptionName(boundaryOption),
                }}
                onChange={(opt) =>
                  setBoundaryOption(boundaryOptionIDFromOption(opt))
                }
              >
                <Select.Button className="h-[30px] py-0">
                  <span className="text-basis">
                    {boundaryOptionName(boundaryOption)}
                  </span>
                </Select.Button>
                <Select.Options>
                  <Select.Option
                    option={{
                      id: 'single_environment',
                      name: boundaryOptionName('single_environment'),
                    }}
                  >
                    Single environment
                  </Select.Option>
                  {envGroups.branchParent && (
                    <Select.Option
                      option={{
                        id: 'all_branch_environments',
                        name: boundaryOptionName('all_branch_environments'),
                      }}
                    >
                      All branch environments
                    </Select.Option>
                  )}
                  <Select.Option
                    option={{
                      id: 'all_environments',
                      name: boundaryOptionName('all_environments'),
                    }}
                  >
                    All environments
                  </Select.Option>
                </Select.Options>
              </Select>
            </div>

            {environmentField}
            {boundaryCallout}
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <label className="text-basis text-sm font-medium">Permissions</label>
          <span className="text-subtle text-xs">
            {selectedResourceCount} resources selected
          </span>
        </div>

        {permissionsContent}
      </div>

      {error && <Alert severity="error">{error}</Alert>}

      <div className="flex justify-end gap-2">
        <Button
          appearance="outlined"
          kind="secondary"
          label="Cancel"
          onClick={cancel}
          disabled={isSubmitting}
        />
        <Button
          kind="primary"
          label="Generate key"
          onClick={submit}
          loading={isSubmitting}
          disabled={isSubmitting}
        />
      </div>
    </div>
  );
}
