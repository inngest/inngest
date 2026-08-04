import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Switch } from '@inngest/components/Switch';
import { useMutation } from 'urql';

import { trackMemberKeyPolicyUpdated } from '@/utils/analyticsEvents';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { apiKeyErrorMessage } from './errorMessage';
import { GrantPicker } from './GrantPicker';
import {
  APIKeyGrantsQuery,
  SetMemberAPIKeyPolicyMutation,
  readOnlyPreset,
} from './grants';

/**
 * MemberKeyPolicyCard is the org-admin control over what members may mint:
 * whether they may mint at all, whether they may target Production, and which
 * permissions they may select.
 *
 * The policy applies at mint time only. Tightening it never alters a key that
 * already exists, so reducing someone's access means revoking their key too —
 * which the copy says out loud, because it is easy to assume otherwise.
 */
export function MemberKeyPolicyCard() {
  const res = useGraphQLQuery({ query: APIKeyGrantsQuery, variables: {} });
  const [, save] = useMutation(SetMemberAPIKeyPolicyMutation);

  const catalog = res.data?.apiKeyGrants ?? [];
  const saved = res.data?.account.memberAPIKeyPolicy;

  // Local edits, discarded on reload. Null means "showing what's saved".
  const [draft, setDraft] = useState<{
    enabled: boolean;
    allowProduction: boolean;
    grants: string[];
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const current = draft ?? saved;
  if (!current) return null;

  // Admins are choosing what to permit, so every grant is selectable here.
  const allGrants = new Set(catalog.map((g) => g.grant));

  function edit(next: Partial<typeof current>) {
    setDraft({ ...current!, ...next });
  }

  async function commit() {
    if (!draft) return;
    setError(null);
    setSaving(true);
    try {
      const result = await save(
        { input: draft },
        { additionalTypenames: ['AccountSetting'] },
      );
      if (result.error) {
        setError(apiKeyErrorMessage(result.error, 'Could not save policy.'));
        return;
      }
      trackMemberKeyPolicyUpdated({
        feature: 'api-keys',
        enabled: draft.enabled,
        allowProduction: draft.allowProduction,
        grantCount: draft.grants.length,
      });
      setDraft(null);
    } finally {
      setSaving(false);
    }
  }

  const dirty = draft !== null;

  return (
    <div className="border-subtle flex flex-col gap-4 rounded-md border p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <span className="text-basis text-sm font-medium">
            Allow members to create API keys
          </span>
          <span className="text-subtle text-sm">
            Members can create API keys from this page or by logging in with the
            Inngest CLI. Admins can always create API keys.
          </span>
        </div>
        <Switch
          checked={current.enabled}
          onCheckedChange={(enabled) => edit({ enabled })}
          disabled={saving || res.isLoading}
        />
      </div>

      {current.enabled && (
        <>
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-basis text-sm font-medium">
                Allow Production keys
              </span>
              <span className="text-subtle text-sm">
                When off, members can only create keys for non-production
                environments.
              </span>
            </div>
            <Switch
              checked={current.allowProduction}
              onCheckedChange={(allowProduction) => edit({ allowProduction })}
              disabled={saving}
            />
          </div>

          <div className="flex flex-col gap-2">
            <span className="text-subtle text-sm">
              Permissions members may select. Defaults to read-only.
            </span>
            <GrantPicker
              grants={catalog}
              selected={current.grants}
              onChange={(grants) => edit({ grants })}
              permitted={allGrants}
              disabled={saving}
            />
            {current.grants.length === 0 && (
              <Alert severity="warning" className="text-sm">
                With no permissions selected, members cannot create a usable
                key.
              </Alert>
            )}
            <div>
              <Button
                appearance="outlined"
                size="small"
                kind="secondary"
                onClick={() => edit({ grants: readOnlyPreset(catalog) })}
                disabled={saving}
                label="Reset to read-only"
              />
            </div>
          </div>
        </>
      )}

      {error && (
        <Alert severity="error" className="text-sm">
          {error}
        </Alert>
      )}

      {dirty && (
        <div className="flex items-center gap-3">
          <Button
            kind="primary"
            size="small"
            onClick={commit}
            loading={saving}
            label="Save policy"
          />
          <Button
            appearance="ghost"
            kind="secondary"
            size="small"
            onClick={() => {
              setDraft(null);
              setError(null);
            }}
            disabled={saving}
            label="Cancel"
          />
          <span className="text-subtle text-xs">
            Applies to new keys only — existing member keys keep their current
            permissions until revoked.
          </span>
        </div>
      )}
    </div>
  );
}
