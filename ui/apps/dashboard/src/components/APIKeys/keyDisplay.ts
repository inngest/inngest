import { EnvironmentType, type GetRestrictedApiKeysQuery } from '@/gql/graphql';

export type APICredential =
  GetRestrictedApiKeysQuery['apiCredentials']['keys'][number];

export function apiKeyStatus(
  key: Pick<APICredential, 'revokedAt' | 'expiresAt'>,
  now = Date.now(),
) {
  if (key.revokedAt) {
    return 'Revoked';
  }
  if (key.expiresAt && new Date(key.expiresAt).getTime() <= now) {
    return 'Expired';
  }
  return 'Active';
}

export function apiKeyEnvironment(key: Pick<APICredential, 'env'>) {
  return key.env?.type === EnvironmentType.BranchParent
    ? 'Branch environments'
    : key.env?.name ?? 'All environments';
}
