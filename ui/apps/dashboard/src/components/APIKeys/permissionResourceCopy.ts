type PermissionResourceCopy = {
  label: string;
  description: string | null;
};

const RESOURCE_COPY: Record<string, PermissionResourceCopy> = {
  accounts: {
    label: 'Accounts',
    description: null,
  },
  api_keys: {
    label: 'API keys',
    description: null,
  },
  apps: {
    label: 'Apps',
    description: null,
  },
  environments: {
    label: 'Environments',
    description: null,
  },
  event_keys: {
    label: 'Event keys',
    description: null,
  },
  events: {
    label: 'Events',
    description: null,
  },
  experiments: {
    label: 'Experiments',
    description: null,
  },
  functions: {
    label: 'Functions',
    description: null,
  },
  insights: {
    label: 'Insights',
    description: null,
  },
  partners: {
    label: 'Partners',
    description: null,
  },
  runs: {
    label: 'Runs',
    description: null,
  },
  sandboxes: {
    label: 'Sandboxes',
    description: null,
  },
  sessions: {
    label: 'Sessions',
    description: null,
  },
  signing_keys: {
    label: 'Signing keys',
    description: null,
  },
  webhooks: {
    label: 'Webhooks',
    description: null,
  },
};

function humanizeResource(resource: string) {
  const words: string[] = [];
  for (const part of resource.split('_')) {
    if (part.length === 0) {
      continue;
    }
    words.push(part.charAt(0).toUpperCase() + part.slice(1));
  }
  return words.join(' ');
}

export function permissionResourceCopy(
  resource: string,
): PermissionResourceCopy {
  const copy = RESOURCE_COPY[resource];
  if (copy) {
    return copy;
  }

  return {
    label: humanizeResource(resource),
    description: null,
  };
}
