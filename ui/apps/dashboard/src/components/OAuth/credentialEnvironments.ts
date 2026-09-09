import type { Option } from '@inngest/components/Select/Select';

import { EnvironmentType, type Environment } from '@/utils/environments';

export function credentialEnvironmentOptions(environments: Environment[]) {
  const production: Option[] = [];
  const test: Option[] = [];
  for (const environment of environments) {
    if (
      environment.isArchived ||
      environment.type === EnvironmentType.BranchChild ||
      environment.type === EnvironmentType.BranchParent
    )
      continue;

    const option = { id: environment.id, name: environment.name };
    if (environment.type === EnvironmentType.Production)
      production.push(option);
    else test.push(option);
  }

  return {
    production,
    test,
    groups: [
      { label: 'Production', opts: production },
      { label: 'Test', opts: test },
    ],
  };
}
