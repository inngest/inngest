import { SecretCheck, type AppCheckResult } from '@/gql/graphql';

export function isAppInfoMissingData(appInfo: AppCheckResult): boolean {
  for (const [k, v] of Object.entries(appInfo)) {
    if (k === 'error') {
      continue;
    }
    if (v === null || v === SecretCheck.Unknown) {
      console.log(k);
      return true;
    }
  }
  return false;
}
