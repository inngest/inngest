import { graphql } from '@/gql';

// The API keys settings page and the device-login page both read this policy;
// the document lives here so graphql-codegen sees a single operation.
export const AllowMemberKeysQuery = graphql(`
  query GetAllowMemberAPIKeysSetting {
    account {
      setting(name: "allow_member_api_keys") {
        value
      }
    }
  }
`);

export const settingQueryContext = { additionalTypenames: ['AccountSetting'] };

// The setting value is a jsonb blob serialized as a JSON string, e.g.
// '{"enabled": true}'. An absent row (null setting) means admins-only.
export function allowMemberKeysEnabled(value: unknown): boolean {
  if (typeof value !== 'string') return false;
  try {
    const parsed: unknown = JSON.parse(value);
    return (
      typeof parsed === 'object' &&
      parsed !== null &&
      (parsed as { enabled?: unknown }).enabled === true
    );
  } catch {
    return false;
  }
}
