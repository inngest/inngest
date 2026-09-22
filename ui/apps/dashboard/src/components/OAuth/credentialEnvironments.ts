import type { Option } from '@inngest/components/Select/Select';

import { EnvironmentType, type Environment } from '@/utils/environments';

export function credentialEnvironmentOptions(
  environments: Environment[],
  { includeBranches = false }: { includeBranches?: boolean } = {},
) {
  const eligible = environments.filter(
    (environment) =>
      !environment.isArchived &&
      environment.type !== EnvironmentType.BranchParent &&
      (includeBranches || environment.type !== EnvironmentType.BranchChild),
  );
  const option = ({ id, name }: Environment): Option => ({ id, name });
  const branches = eligible
    .filter((env) => env.type === EnvironmentType.BranchChild)
    .map(option);
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
        .filter((env) => env.type === EnvironmentType.Test)
        .map(option),
    },
    ...(branches.length > 0
      ? [
          {
            label: 'Branches',
            opts: branches,
          },
        ]
      : []),
  ];
}
