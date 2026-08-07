import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill/Pill';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiArrowRightSLine,
  RiCheckLine,
  RiDeleteBin6Line,
  RiLockLine,
  RiPencilLine,
  RiSubtractLine,
} from '@remixicon/react';

import { groupGrants, type Grant } from './grants';

export type APIKeyRow = {
  id: string;
  name: string;
  maskedKey: string;
  createdAt: string;
  env: { id: string; name: string } | null;
  envs: { id: string; name: string }[];
  envScope: string;
  scopes: { name: string; allow: string[] }[];
  /** Resolved server-side, so legacy scopes are already translated out. */
  grants: string[];
  /** Read with admin status to decide whether the viewer may manage the key. */
  createdByViewer: boolean;
  // Null for pre-attribution keys, machine-provisioned keys, and keys whose
  // creator was deleted.
  createdBy: { name: string | null; email: string } | null;
};

type Props = {
  keys: APIKeyRow[];
  /** Full catalog, so every key's grid shows the same rows in the same order. */
  catalog: Grant[];
  isAdmin: boolean;
  onRename: (key: APIKeyRow) => void;
  onDelete: (key: APIKeyRow) => void;
};

/**
 * A key predating permission grants holds only the retired `api:app:sync`
 * scope. It is shown as stored rather than translated, because translating it
 * would hide the one thing worth knowing: re-minting is what gives the key real
 * permissions.
 */
function isLegacyKey(scopes: { name: string }[] | undefined): boolean {
  if (!scopes || scopes.length === 0) return false;
  return scopes.every((s) => s.name === 'api:app:sync');
}

function creatorName(key: APIKeyRow): string {
  return key.createdBy?.name ?? key.createdBy?.email ?? 'Unknown';
}

function envSummary(key: APIKeyRow): string {
  if (key.envScope === 'account') return 'All environments';
  if (key.envs.length > 0) return key.envs.map((e) => e.name).join(', ');
  return key.env?.name ?? '—';
}

const COLUMNS = 'grid-cols-[24px_1.5fr_1.3fr_1fr_0.8fr_124px]';

export function APIKeysTable({
  keys,
  catalog,
  isAdmin,
  onRename,
  onDelete,
}: Props) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const groups = groupGrants(catalog);

  return (
    <div className="border-subtle overflow-hidden rounded-md border">
      <div
        className={cn(
          COLUMNS,
          'bg-canvasSubtle text-muted grid gap-4 px-3.5 py-2 text-[11px] font-semibold uppercase tracking-wide',
        )}
      >
        <span />
        <span>Name</span>
        <span>Key</span>
        <span>Created by</span>
        <span>Created</span>
        <span />
      </div>

      {keys.map((key) => {
        const open = expanded === key.id;
        const canManage = isAdmin || key.createdByViewer;
        const granted = new Set(key.grants);
        return (
          <div key={key.id} className="border-subtle border-t">
            <div
              className={cn(
                COLUMNS,
                'grid items-center gap-4 px-3.5 py-2.5 text-sm',
                open && 'bg-canvasSubtle',
              )}
            >
              <button
                type="button"
                aria-label={open ? 'Collapse' : 'Expand'}
                aria-expanded={open}
                className="text-muted flex h-5 w-5 items-center justify-center"
                onClick={() => setExpanded(open ? null : key.id)}
              >
                <RiArrowRightSLine
                  className={cn(
                    'h-4 w-4 transition-transform',
                    open && 'rotate-90',
                  )}
                />
              </button>

              <div className="flex min-w-0 flex-col">
                <div className="flex items-center gap-2">
                  <span className="text-basis truncate">{key.name}</span>
                  {isLegacyKey(key.scopes) && (
                    <Pill appearance="outlined" kind="warning">
                      Legacy
                    </Pill>
                  )}
                </div>
                {isLegacyKey(key.scopes) && (
                  <span className="text-light text-xs">
                    Predates permissions, so it can only sync apps. Create a new
                    key to choose what it can access.
                  </span>
                )}
              </div>

              <span className="text-subtle truncate font-mono text-xs">
                {key.maskedKey}
              </span>
              <span
                className={cn(
                  'truncate',
                  key.createdByViewer ? 'text-basis' : 'text-subtle',
                )}
              >
                {creatorName(key)}
              </span>
              <Time
                className="text-subtle text-sm"
                format="relative"
                value={key.createdAt}
              />

              {canManage ? (
                <span className="flex justify-end gap-2">
                  <Button
                    appearance="outlined"
                    kind="secondary"
                    size="small"
                    icon={<RiPencilLine />}
                    label="Rename"
                    onClick={() => onRename(key)}
                  />
                  <Button
                    appearance="outlined"
                    kind="danger"
                    size="small"
                    icon={<RiDeleteBin6Line />}
                    onClick={() => onDelete(key)}
                    aria-label="Revoke"
                  />
                </span>
              ) : (
                <RiLockLine
                  className="text-disabled h-4 w-4 justify-self-end"
                  aria-label="You cannot manage this key"
                />
              )}
            </div>

            {open && (
              <div className="bg-canvasSubtle border-subtle flex flex-col gap-3 border-t py-3.5 pl-[52px] pr-3.5">
                <div className="flex flex-wrap items-baseline gap-3">
                  <span className="text-muted text-[11px] font-semibold uppercase tracking-wide">
                    Environments
                  </span>
                  <span className="text-subtle text-xs">{envSummary(key)}</span>
                </div>

                <div className="flex flex-wrap items-baseline gap-3">
                  <span className="text-muted text-[11px] font-semibold uppercase tracking-wide">
                    Permissions
                  </span>
                  <span className="text-subtle font-mono text-xs">
                    {key.grants.length} of {catalog.length} grants allowed
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-x-6 gap-y-3.5 md:grid-cols-4">
                  {groups.map((group) => (
                    <div key={group.category} className="flex flex-col gap-1">
                      <span className="text-light text-[10px] font-semibold uppercase tracking-wider">
                        {group.category}
                      </span>
                      {group.grants.map((g) => {
                        const on = granted.has(g.grant);
                        return (
                          <div
                            key={g.grant}
                            className="flex items-center gap-1.5"
                          >
                            {on ? (
                              <RiCheckLine className="text-success h-3 w-3 shrink-0" />
                            ) : (
                              <RiSubtractLine className="text-disabled h-3 w-3 shrink-0" />
                            )}
                            <span
                              className={cn(
                                'font-mono text-[11px]',
                                on ? 'text-subtle' : 'text-disabled',
                              )}
                            >
                              {g.grant}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                  ))}
                </div>

                {canManage ? (
                  <div>
                    <Button
                      appearance="outlined"
                      kind="danger"
                      size="small"
                      label="Revoke key"
                      onClick={() => onDelete(key)}
                    />
                  </div>
                ) : (
                  <span className="text-light flex items-center gap-1.5 text-xs">
                    <RiLockLine className="h-3 w-3 shrink-0" />
                    Only {creatorName(key)} or an admin can revoke this key.
                  </span>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
