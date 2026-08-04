import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Switch } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import { useMutation } from 'urql';

import { trackMemberKeyPolicyUpdated } from '@/utils/analyticsEvents';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import { apiKeyErrorMessage } from './errorMessage';
import {
  APIKeyGrantsQuery,
  SetMemberAPIKeyPolicyMutation,
  activePreset,
  fullAccessPreset,
  groupGrants,
  readOnlyPreset,
} from './grants';

/**
 * MemberKeyPolicySidebar is the org-admin control over what members may mint:
 * whether they may mint at all, whether they may target Production, and which
 * permissions they may select.
 *
 * The policy applies at mint time only. Tightening it never alters a key that
 * already exists, so reducing someone's access means revoking their key too —
 * which the copy says out loud, because it is easy to assume otherwise.
 *
 * Each grant is an Allowed/Disallowed choice rather than a switch, deliberately:
 * this list is what members *may* have, not what any key *does* have, and the two
 * read identically when both are toggles.
 */
export function MemberKeyPolicySidebar() {
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
  const groups = groupGrants(catalog);
  const allowed = new Set(current.grants);
  const preset = activePreset(current.grants, catalog, allGrants);

  function edit(next: Partial<typeof current>) {
    setDraft({ ...current!, ...next });
  }

  function setAllowed(grant: string, next: boolean) {
    const remaining = current!.grants.filter((g) => g !== grant);
    edit({ grants: next ? [...remaining, grant].sort() : remaining });
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
    <aside className="border-subtle bg-canvasSubtle sticky top-0 flex max-h-screen w-[432px] shrink-0 flex-col rounded-md border">
      <div className="border-subtle flex flex-col gap-3 border-b p-5">
        <h2 className="text-basis text-sm font-semibold">Member key policy</h2>

        <div className="flex items-start gap-3">
          <div className="flex flex-1 flex-col gap-0.5">
            <span className="text-basis text-[13px] font-medium">
              Allow members to create API keys
            </span>
            <span className="text-subtle text-xs">
              When off, only admins can create keys. Existing member keys keep
              working.
            </span>
          </div>
          <Switch
            checked={current.enabled}
            onCheckedChange={(enabled) => edit({ enabled })}
            disabled={saving || res.isLoading}
          />
        </div>

        {current.enabled && (
          <div className="border-subtle flex items-start gap-3 border-t pt-3">
            <div className="flex flex-1 flex-col gap-0.5">
              <div className="flex items-center gap-1.5">
                <span className="text-basis text-[13px] font-medium">
                  Allow the Production environment
                </span>
                <span className="bg-success h-1.5 w-1.5 shrink-0 rounded-full" />
              </div>
              <span className="text-subtle text-xs">
                When off, member keys can only select branch and non-production
                environments.
              </span>
            </div>
            <Switch
              checked={current.allowProduction}
              onCheckedChange={(allowProduction) => edit({ allowProduction })}
              disabled={saving}
            />
          </div>
        )}
      </div>

      {current.enabled && (
        <>
          <div className="border-subtle flex flex-col gap-2.5 border-b px-5 py-3.5">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-basis text-[13px] font-semibold">
                Permissions members may grant
              </span>
              <span className="text-subtle font-mono text-xs">
                {current.grants.length} of {catalog.length} allowed
              </span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {(
                [
                  ['Read only', 'readOnly', readOnlyPreset],
                  ['Full access', 'fullAccess', fullAccessPreset],
                ] as const
              ).map(([label, name, build]) => (
                <button
                  key={name}
                  type="button"
                  disabled={saving}
                  className={cn(
                    'flex h-6 items-center rounded-2xl border px-2.5 text-xs font-medium',
                    preset === name
                      ? 'border-success bg-success text-success'
                      : 'border-subtle text-subtle',
                  )}
                  onClick={() => edit({ grants: build(catalog) })}
                >
                  {label}
                </button>
              ))}
              <span
                className={cn(
                  'flex h-6 items-center rounded-2xl border px-2.5 text-xs font-medium',
                  preset === 'custom'
                    ? 'border-muted bg-canvasBase text-basis'
                    : 'border-subtle text-disabled',
                )}
              >
                Custom
              </span>
            </div>
          </div>

          <div className="flex-1 overflow-auto">
            {groups.map((group) => (
              <div key={group.category}>
                <div className="border-subtle bg-canvasBase sticky top-0 flex items-center justify-between border-b px-5 py-1.5">
                  <span className="text-muted text-[10px] font-semibold uppercase tracking-wider">
                    {group.category}
                  </span>
                  <span className="text-light font-mono text-[10px]">
                    {group.grants.filter((g) => allowed.has(g.grant)).length}{' '}
                    allowed
                  </span>
                </div>
                {group.grants.map((g) => {
                  const on = allowed.has(g.grant);
                  return (
                    <div
                      key={g.grant}
                      className="border-subtle flex items-center gap-3 border-b px-5 py-2"
                    >
                      <div className="flex min-w-0 flex-1 flex-col">
                        <span className="text-basis font-mono text-xs">
                          {g.grant}
                        </span>
                        <span className="text-subtle text-[11px]">
                          {g.description}
                        </span>
                      </div>
                      <select
                        aria-label={g.grant}
                        disabled={saving}
                        value={on ? 'allowed' : 'disallowed'}
                        onChange={(e) =>
                          setAllowed(g.grant, e.target.value === 'allowed')
                        }
                        className={cn(
                          'border-muted bg-canvasBase h-[26px] w-[118px] shrink-0 rounded-md border px-2 text-[11.5px] font-medium',
                          on ? 'text-basis' : 'text-subtle',
                        )}
                      >
                        <option value="allowed">Allowed</option>
                        <option value="disallowed">Disallowed</option>
                      </select>
                    </div>
                  );
                })}
              </div>
            ))}
          </div>

          {current.grants.length === 0 && (
            <Alert severity="warning" className="mx-5 my-3 text-xs">
              With no permissions allowed, members cannot create a usable key.
            </Alert>
          )}
        </>
      )}

      {error && (
        <Alert severity="error" className="mx-5 my-3 text-xs">
          {error}
        </Alert>
      )}

      <div className="border-subtle flex items-center justify-between gap-3 border-t p-4">
        <span className="text-subtle text-xs">
          {dirty
            ? 'Applies to new keys only — existing member keys keep their permissions until revoked.'
            : 'Members choose from the allowed grants only.'}
        </span>
        <div className="flex shrink-0 gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
            size="small"
            label="Reset"
            disabled={saving || !dirty}
            onClick={() => {
              setDraft(null);
              setError(null);
            }}
          />
          <Button
            kind="primary"
            size="small"
            label="Save"
            loading={saving}
            disabled={saving || !dirty}
            onClick={commit}
          />
        </div>
      </div>
    </aside>
  );
}
