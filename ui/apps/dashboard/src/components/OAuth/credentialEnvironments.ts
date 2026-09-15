import type { Option } from '@inngest/components/Select/Select';

import { EnvironmentType, type Environment } from '@/utils/environments';

export function credentialEnvironmentOptions(environments: Environment[]) {
  const eligible = environments.filter(
    (environment) =>
      !environment.isArchived &&
      environment.type !== EnvironmentType.BranchChild &&
      environment.type !== EnvironmentType.BranchParent,
  );
  const option = ({ id, name }: Environment): Option => ({ id, name });
  return [
    {
      label: 'Production',
      opts: eligible
        .filter((env) => env.type === EnvironmentType.Production)
        .map(option),
    },
    {
      label: 'Test',
      opts: eligible
        .filter((env) => env.type !== EnvironmentType.Production)
        .map(option),
    },
  ];
}
