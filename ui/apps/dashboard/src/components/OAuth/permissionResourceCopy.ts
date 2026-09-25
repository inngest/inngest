type PermissionResourceCopy = {
  label: string;
  description: string | null;
};

const RESOURCE_LABELS: Record<string, string> = {
  accounts: 'Accounts',
  api_keys: 'API keys',
  apps: 'Apps',
  environments: 'Environments',
  event_keys: 'Event keys',
  events: 'Events',
  experiments: 'Experiments',
  functions: 'Functions',
  insights: 'Insights',
  partners: 'Partners',
  runs: 'Runs',
  sandboxes: 'Sandboxes',
  sessions: 'Sessions',
  signing_keys: 'Signing keys',
  webhooks: 'Webhooks',
};

function humanizeResource(resource: string) {
  return resource
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function permissionResourceCopy(
  resource: string,
): PermissionResourceCopy {
  return {
    label: RESOURCE_LABELS[resource] ?? humanizeResource(resource),
    description:
      resource === 'webhooks'
        ? 'Read access includes URLs that can send events. Copied URLs still work after key or session revocation. Revoke the webhook to disable them.'
        : null,
  };
}
