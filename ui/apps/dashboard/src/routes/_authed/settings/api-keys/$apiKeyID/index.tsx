import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Time } from '@inngest/components/Time';
import { RiArrowLeftLine } from '@remixicon/react';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { gql, useQuery, type TypedDocumentNode } from 'urql';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import {
  ApiKeyCredentialSource,
  ApiKeyResourceBoundaryMode,
  EnvironmentType,
} from '@/gql/graphql';

export const Route = createFileRoute('/_authed/settings/api-keys/$apiKeyID/')({
  component: APIKeyPage,
});

type APIKeyScope = {
  name: string;
  allow: string[];
  deny: string[];
};

type APIKeyPermissionGroup = {
  resource: string;
  label: string;
  description: string | null;
  read: string[];
  write: string[];
};

type APIKeyDetail = {
  id: string;
  name: string;
  createdAt: string;
  resourceBoundaryMode: ApiKeyResourceBoundaryMode;
  expiresAt: string | null;
  credentialSource: ApiKeyCredentialSource;
  maskedKey: string;
  env: { id: string; name: string; type: EnvironmentType } | null;
  permissionGrants: string[];
  scopes: APIKeyScope[];
};

type APIKeyDetailResult = {
  account: {
    apiKey: APIKeyDetail;
  };
  apiKeyPermissionCatalog: APIKeyPermissionGroup[];
};

type APIKeyDetailVariables = {
  id: string;
};

type PermissionLevel = 'None' | 'Read' | 'Write';

const Query: TypedDocumentNode<APIKeyDetailResult, APIKeyDetailVariables> = gql`
  query GetAPIKeyDetail($id: UUID!) {
    account {
      apiKey(id: $id) {
        id
        name
        createdAt
        resourceBoundaryMode
        expiresAt
        credentialSource
        maskedKey
        env {
          id
          name
          type
        }
        permissionGrants
        scopes {
          name
          allow
          deny
        }
      }
    }
    apiKeyPermissionCatalog {
      resource
      label
      description
      read
      write
    }
  }
`;

function readonlyFieldClass() {
  return 'border-subtle bg-canvasSubtle text-basis flex h-8 items-center rounded border px-3 text-sm';
}

function permissionLevelClass(level: PermissionLevel) {
  const classes = [
    'inline-flex h-7 min-w-16 items-center justify-center rounded-full px-3 text-sm',
  ];
  if (level === 'None') {
    classes.push('bg-canvasMuted text-muted');
  } else {
    classes.push('bg-canvasSubtle border-muted text-basis border');
  }
  return classes.join(' ');
}

function isAllBranchEnvironmentsBoundary(key: APIKeyDetail) {
  if (key.resourceBoundaryMode !== ApiKeyResourceBoundaryMode.SingleEnv) {
    return false;
  }
  if (!key.env) {
    return false;
  }
  if (key.env.type !== EnvironmentType.BranchParent) {
    return false;
  }
  return true;
}

function boundaryModeName(key: APIKeyDetail) {
  if (isAllBranchEnvironmentsBoundary(key)) {
    return 'All branch environments';
  }
  if (key.resourceBoundaryMode === ApiKeyResourceBoundaryMode.AllEnvs) {
    return 'All environments';
  }
  return 'Single environment';
}

function credentialSourceName(credentialSource: ApiKeyCredentialSource) {
  switch (credentialSource) {
    case ApiKeyCredentialSource.CliLogin:
      return 'CLI login';
    case ApiKeyCredentialSource.DashboardUi:
      return 'Dashboard UI';
    case ApiKeyCredentialSource.Legacy:
      return 'Legacy';
    case ApiKeyCredentialSource.McpLogin:
      return 'MCP login';
  }
}

function shouldShowEnvironment(key: APIKeyDetail) {
  if (key.resourceBoundaryMode !== ApiKeyResourceBoundaryMode.SingleEnv) {
    return false;
  }
  if (isAllBranchEnvironmentsBoundary(key)) {
    return false;
  }
  if (!key.env) {
    return false;
  }
  return true;
}

function shouldShowBoundaryCallout(key: APIKeyDetail) {
  if (isAllBranchEnvironmentsBoundary(key)) {
    return true;
  }
  if (key.resourceBoundaryMode === ApiKeyResourceBoundaryMode.AllEnvs) {
    return true;
  }
  return false;
}

function permissionLevelForGroup(
  group: APIKeyPermissionGroup,
  scopeNames: Set<string>,
): PermissionLevel {
  for (const grant of group.write) {
    if (scopeNames.has(grant)) {
      return 'Write';
    }
  }
  for (const grant of group.read) {
    if (scopeNames.has(grant)) {
      return 'Read';
    }
  }
  return 'None';
}

function catalogScopeNames(groups: APIKeyPermissionGroup[]) {
  const names = new Set<string>();
  for (const group of groups) {
    for (const grant of group.read) {
      names.add(grant);
    }
    for (const grant of group.write) {
      names.add(grant);
    }
  }
  return names;
}

function unrecognizedScopes(
  scopes: APIKeyScope[],
  recognizedScopeNames: Set<string>,
) {
  const unrecognized: APIKeyScope[] = [];
  for (const scope of scopes) {
    if (recognizedScopeNames.has(scope.name)) {
      continue;
    }
    unrecognized.push(scope);
  }
  return unrecognized;
}

function unrecognizedPermissionGrants(
  permissionGrants: string[],
  recognizedScopeNames: Set<string>,
) {
  const unrecognized: string[] = [];
  for (const grant of permissionGrants) {
    if (recognizedScopeNames.has(grant)) {
      continue;
    }
    unrecognized.push(grant);
  }
  return unrecognized;
}

function actionList(actions: string[]) {
  if (actions.length === 0) {
    return '-';
  }
  return actions.join(', ');
}

function APIKeyPage() {
  const navigate = useNavigate();
  const { apiKeyID } = Route.useParams();
  const [{ data, error, fetching }] = useQuery({
    query: Query,
    requestPolicy: 'cache-first',
    variables: { id: apiKeyID },
  });

  const backToList = () => navigate({ to: '/settings/api-keys' });

  if (error) {
    return (
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-8">
        <APIKeyHeader title="API key" onBack={backToList} />
        <Alert severity="error">Could not load API key.</Alert>
      </div>
    );
  }
  if (fetching && !data) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (!data) {
    return (
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-8">
        <APIKeyHeader title="API key" onBack={backToList} />
        <Alert severity="error">API key not found.</Alert>
      </div>
    );
  }

  const apiKey = data.account.apiKey;

  const permissionGrantNames = new Set<string>();
  for (const grant of apiKey.permissionGrants) {
    permissionGrantNames.add(grant);
  }

  const grantedPermissionGroups: {
    group: APIKeyPermissionGroup;
    level: PermissionLevel;
  }[] = [];
  for (const group of data.apiKeyPermissionCatalog) {
    const level = permissionLevelForGroup(group, permissionGrantNames);
    if (level === 'None') {
      continue;
    }
    grantedPermissionGroups.push({ group, level });
  }
  const recognizedScopeNames = catalogScopeNames(data.apiKeyPermissionCatalog);
  const additionalGrants = unrecognizedPermissionGrants(
    apiKey.permissionGrants,
    recognizedScopeNames,
  );
  const legacyScopes = unrecognizedScopes(apiKey.scopes, new Set<string>());

  let permissionsTable = null;
  if (grantedPermissionGroups.length > 0) {
    permissionsTable = (
      <div className="border-subtle rounded border">
        {grantedPermissionGroups.map(({ group, level }) => {
          return (
            <div
              key={group.resource}
              className="border-subtle grid grid-cols-1 gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
            >
              <div className="flex min-w-0 flex-col">
                <span className="text-basis truncate text-sm font-medium">
                  {group.label}
                </span>
                {group.description && (
                  <span className="text-subtle truncate text-xs">
                    {group.description}
                  </span>
                )}
              </div>

              <span className={permissionLevelClass(level)}>{level}</span>
            </div>
          );
        })}
      </div>
    );
  }

  let expiresAt = <span>No expiration</span>;
  if (apiKey.expiresAt) {
    expiresAt = <Time value={apiKey.expiresAt} />;
  }

  let environmentField = null;
  if (shouldShowEnvironment(apiKey) && apiKey.env) {
    environmentField = (
      <div className="flex flex-col gap-1">
        <label className="text-basis text-sm font-medium">Environment</label>
        <div className={readonlyFieldClass()}>{apiKey.env.name}</div>
      </div>
    );
  }

  let boundaryCallout = null;
  if (shouldShowBoundaryCallout(apiKey)) {
    let boundaryCalloutText =
      'API requests made with this key must specify the environment name.';
    if (isAllBranchEnvironmentsBoundary(apiKey)) {
      boundaryCalloutText =
        'This key applies to every current and future branch environment. API requests made with this key must specify the branch environment name.';
    }

    boundaryCallout = (
      <div className="bg-canvasSubtle border-subtle text-subtle rounded border px-3 py-2 text-sm sm:col-span-2">
        {boundaryCalloutText}
      </div>
    );
  }

  let additionalPermissions = null;
  if (additionalGrants.length > 0) {
    additionalPermissions = (
      <div className="flex flex-col gap-3">
        <label className="text-basis text-sm font-medium">
          Additional permissions
        </label>
        <div className="border-subtle rounded border">
          {additionalGrants.map((grant) => {
            return (
              <div
                key={grant}
                className="border-subtle grid grid-cols-1 gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
              >
                <div className="flex min-w-0 flex-col">
                  <span className="text-basis truncate font-mono text-sm">
                    {grant}
                  </span>
                  <span className="text-subtle truncate text-xs">
                    Raw permission not yet mapped to dashboard copy.
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  let legacyPermissions = null;
  if (legacyScopes.length > 0) {
    legacyPermissions = (
      <div className="flex flex-col gap-3">
        <label className="text-basis text-sm font-medium">
          Legacy permissions
        </label>
        <div className="border-subtle rounded border">
          {legacyScopes.map((scope) => {
            return (
              <div
                key={scope.name}
                className="border-subtle grid grid-cols-1 gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
              >
                <div className="flex min-w-0 flex-col">
                  <span className="text-basis truncate font-mono text-sm">
                    {scope.name}
                  </span>
                  <span className="text-subtle truncate text-xs">
                    Compatibility permission stored in the legacy scopes field.
                  </span>
                </div>

                <div className="flex flex-col gap-1 text-right font-mono text-xs">
                  <span className="text-subtle">
                    allow: {actionList(scope.allow)}
                  </span>
                  <span className="text-subtle">
                    deny: {actionList(scope.deny)}
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-8">
      <APIKeyHeader title={apiKey.name} onBack={backToList} />

      <div className="flex w-full flex-col gap-6">
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">
              API key name
            </label>
            <div className={readonlyFieldClass()}>{apiKey.name}</div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">Expiration</label>
            <div className={readonlyFieldClass()}>{expiresAt}</div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">Created</label>
            <div className={readonlyFieldClass()}>
              <Time format="relative" value={apiKey.createdAt} />
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">Source</label>
            <div className={readonlyFieldClass()}>
              {credentialSourceName(apiKey.credentialSource)}
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-basis text-sm font-medium">API key</label>
            <div className="border-subtle bg-canvasSubtle text-basis flex h-8 items-center rounded border px-3 font-mono text-sm">
              {apiKey.maskedKey}
            </div>
          </div>

          <div className="flex flex-col gap-3 sm:col-span-2">
            <label className="text-basis text-sm font-medium">Boundary</label>
            <div className="border-subtle grid grid-cols-1 gap-5 rounded border p-3 sm:grid-cols-2">
              <div className="flex flex-col gap-1">
                <label className="text-basis text-sm font-medium">Mode</label>
                <div className={readonlyFieldClass()}>
                  {boundaryModeName(apiKey)}
                </div>
              </div>

              {environmentField}
              {boundaryCallout}
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-3">
          <label className="text-basis text-sm font-medium">Permissions</label>
          {permissionsTable}
        </div>

        {additionalPermissions}
        {legacyPermissions}
      </div>
    </div>
  );
}

function APIKeyHeader({
  title,
  onBack,
}: {
  title: string;
  onBack: () => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="text-basis truncate text-2xl">{title}</h1>
      </div>
      <Button
        appearance="outlined"
        kind="secondary"
        icon={<RiArrowLeftLine />}
        iconSide="left"
        label="Back"
        onClick={onBack}
      />
    </div>
  );
}
