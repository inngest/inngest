import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { Switch } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import { RiCheckLine, RiCloseLine } from '@remixicon/react';
import { useMutation } from 'urql';

import LoadingIcon from '@/components/Icons/LoadingIcon';
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

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

/**
 * MemberKeyPolicyModal is the org-admin control over what members may mint:
 * whether they may mint at all, whether they may target Production, and which
 * permissions they may select.
 *
 * The policy applies at mint time only. Tightening it never alters a key that
 * already exists, so reducing someone's access means revoking their key too —
 * which the copy says out loud, because it is easy to assume otherwise.
 *
 * Each grant is an Allowed/Blocked choice rather than a switch, deliberately:
 * this list is what members *may* have, not what any key *does* have, and the two
 * read identically when both are toggles.
 */
export function MemberKeyPolicyModal({ isOpen, onClose }: Props) {
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

  // Admins are choosing what to permit, so every grant is selectable here.
  const allGrants = new Set(catalog.map((g) => g.grant));
  const groups = groupGrants(catalog);
  const allowed = new Set(current?.grants ?? []);
  const preset = activePreset(current?.grants ?? [], catalog, allGrants);

  function edit(next: Partial<NonNullable<typeof current>>) {
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
    <Modal
      className="flex max-h-[78vh] w-full max-w-2xl flex-col overflow-y-hidden"
      isOpen={isOpen}
      onClose={onClose}
    >
      <Modal.Header description="Applies to every key a non-admin member creates.">
        Member key policy
      </Modal.Header>

      {!current ? (
        <div className="flex h-40 w-full items-center justify-center">
          <LoadingIcon />
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="border-subtle flex flex-col gap-3 border-b p-6">
            <div className="flex items-start gap-3">
              <div className="flex flex-1 flex-col gap-0.5">
                <span className="text-basis text-[13px] font-medium">
                  Allow members to create API keys
                </span>
                <span className="text-subtle text-xs">
                  When off, only admins can create keys. Existing member keys
                  keep working.
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
                    When off, member keys can only select branch and
                    non-production environments.
                  </span>
                </div>
                <Switch
                  checked={current.allowProduction}
                  onCheckedChange={(allowProduction) =>
                    edit({ allowProduction })
                  }
                  disabled={saving}
                />
              </div>
            )}
          </div>

          {current.enabled && (
            <>
              <div className="border-subtle flex flex-col gap-2.5 border-b px-6 py-3.5">
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
                  {/* Custom is arrived at, not chosen — it lights up when the
                      selection stops matching a preset, so there is nothing to
                      apply when it is not already active. */}
                  <button
                    type="button"
                    disabled={preset !== 'custom'}
                    className={cn(
                      'flex h-6 items-center rounded-2xl border px-2.5 text-xs font-medium',
                      preset === 'custom'
                        ? 'border-muted bg-canvasBase text-basis'
                        : 'border-subtle text-disabled',
                    )}
                  >
                    Custom
                  </button>
                </div>
              </div>

              <div>
                {groups.map((group) => (
                  <div key={group.category}>
                    <div className="border-subtle bg-canvasSubtle sticky top-0 flex items-center justify-between border-b px-6 py-1.5">
                      <span className="text-muted text-[10px] font-semibold uppercase tracking-wider">
                        {group.category}
                      </span>
                      <span className="text-light font-mono text-[10px]">
                        {
                          group.grants.filter((g) => allowed.has(g.grant))
                            .length
                        }{' '}
                        of {group.grants.length}
                      </span>
                    </div>
                    {group.grants.map((g) => {
                      const on = allowed.has(g.grant);
                      return (
                        <button
                          key={g.grant}
                          type="button"
                          aria-label={g.grant}
                          aria-pressed={on}
                          disabled={saving}
                          onClick={() => setAllowed(g.grant, !on)}
                          className="border-subtle hover:bg-canvasSubtle flex w-full items-center gap-4 border-b px-6 py-2.5 text-left"
                        >
                          <div className="flex min-w-0 flex-1 flex-col">
                            <span className="text-basis font-mono text-xs">
                              {g.grant}
                            </span>
                            <span className="text-subtle text-[11px]">
                              {g.description}
                            </span>
                          </div>
                          <span
                            className={cn(
                              'flex h-[26px] shrink-0 items-center gap-1.5 rounded-2xl border px-2.5 text-[11.5px]',
                              on
                                ? 'border-success bg-success text-success font-semibold'
                                : 'border-subtle text-subtle font-medium',
                            )}
                          >
                            {on ? (
                              <RiCheckLine className="h-3.5 w-3.5" />
                            ) : (
                              <RiCloseLine className="h-3.5 w-3.5" />
                            )}
                            {on ? 'Allowed' : 'Blocked'}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                ))}
              </div>

              {current.grants.length === 0 && (
                <Alert severity="warning" className="mx-6 my-3 text-xs">
                  With no permissions allowed, members cannot create a usable
                  key.
                </Alert>
              )}
            </>
          )}

          {error && (
            <Alert severity="error" className="mx-6 my-3 text-xs">
              {error}
            </Alert>
          )}
        </div>
      )}

      <Modal.Footer className="px-6 py-4">
        <div className="flex items-center justify-between gap-3">
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
      </Modal.Footer>
    </Modal>
  );
}
