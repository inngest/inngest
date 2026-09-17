import { describe, expect, it } from 'vitest';

import { EnvironmentType, type Environment } from '@/utils/environments';
import { credentialEnvironmentOptions } from './credentialEnvironments';

describe('credential environments', () => {
  it('groups active production and test environments, excluding branches', () => {
    const environment = (
      id: string,
      type: EnvironmentType,
      isArchived = false,
    ): Environment => ({
      id,
      name: id,
      type,
      isArchived,
      slug: id,
      hasParent: false,
      webhookSigningKey: '',
      createdAt: '',
      isAutoArchiveEnabled: false,
      lastDeployedAt: null,
    });
    expect(
      credentialEnvironmentOptions([
        environment('prod', EnvironmentType.Production),
        environment('test', EnvironmentType.Test),
        environment('archived', EnvironmentType.Test, true),
        environment('branch', EnvironmentType.BranchChild),
        environment('parent', EnvironmentType.BranchParent),
      ]),
    ).toEqual([
      { label: 'Production', opts: [{ id: 'prod', name: 'prod' }] },
      { label: 'Test', opts: [{ id: 'test', name: 'test' }] },
    ]);
  });
});
