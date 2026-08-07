import { Switch } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import { RiCheckLine, RiLockLine, RiSubtractLine } from '@remixicon/react';

import {
  activePreset,
  fullAccessPreset,
  groupGrants,
  readOnlyPreset,
  type Grant,
} from './grants';

type Props = {
  grants: Grant[];
  selected: string[];
  onChange: (grants: string[]) => void;
  /**
   * Anything outside this renders locked rather than hidden, so a member can
   * see that a permission exists and that an admin controls it.
   */
  permitted: Set<string>;
  /** Shown once above the list when some grants are locked. */
  restrictionNote?: string;
  summaryNote?: string;
  disabled?: boolean;
};

function chipClass(active: boolean): string {
  return cn(
    'flex h-6 items-center rounded-2xl border px-2.5 text-xs font-medium',
    active
      ? 'border-success bg-success text-success'
      : 'border-subtle text-subtle',
  );
}

export function GrantPicker({
  grants,
  selected,
  onChange,
  permitted,
  restrictionNote,
  summaryNote,
  disabled = false,
}: Props) {
  const groups = groupGrants(grants);
  const selectedSet = new Set(selected);
  const preset = activePreset(selected, grants, permitted);

  // Locked rows change the list's shape: every row gains a state icon and the
  // group counts switch to "how many you may select". Derived from the data
  // rather than a mode flag so the two cannot disagree.
  const restricted = grants.some((g) => !permitted.has(g.grant));

  function toggle(grant: string, checked: boolean) {
    const next = new Set(selectedSet);
    if (checked) {
      next.add(grant);
    } else {
      next.delete(grant);
    }
    onChange([...next].sort());
  }

  function applyPreset(grantsInPreset: string[]) {
    // A member clicking Full access gets the most they are allowed rather than
    // an error.
    onChange(grantsInPreset.filter((g) => permitted.has(g)).sort());
  }

  const total = grants.length;
  const allowedCount = grants.filter((g) => permitted.has(g.grant)).length;
  const summary = restricted
    ? `Your admins allow ${allowedCount} of ${total} grants · ${selected.length} selected`
    : `${selected.length} of ${total} grants selected`;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-basis text-sm font-medium">Permissions</span>
        <span className="text-light font-mono text-xs">
          {summaryNote ? `${summary} · ${summaryNote}` : summary}
        </span>
      </div>

      <div className="flex gap-1.5">
        <button
          type="button"
          disabled={disabled}
          className={chipClass(preset === 'readOnly')}
          onClick={() => applyPreset(readOnlyPreset(grants))}
        >
          Read only
        </button>
        <button
          type="button"
          disabled={disabled}
          className={chipClass(preset === 'fullAccess')}
          onClick={() => applyPreset(fullAccessPreset(grants))}
        >
          Full access
        </button>
        {/* Custom is derived, not a choice: it reports that the selection no
            longer matches a preset. */}
        <span
          className={cn(
            chipClass(false),
            preset === 'custom'
              ? 'border-muted bg-canvasSubtle text-basis'
              : 'text-disabled',
          )}
        >
          Custom
        </span>
      </div>

      {restricted && restrictionNote && (
        <div className="border-subtle bg-canvasSubtle text-light flex items-center gap-1.5 rounded-md border px-2.5 py-2 text-xs">
          <RiLockLine className="text-muted h-3.5 w-3.5 shrink-0" />
          {restrictionNote}
        </div>
      )}

      <div className="border-subtle max-h-96 overflow-auto rounded-md border">
        {groups.map((group) => {
          const count = restricted
            ? `${
                group.grants.filter((g) => permitted.has(g.grant)).length
              } of ${group.grants.length} allowed`
            : `${
                group.grants.filter((g) => selectedSet.has(g.grant)).length
              } of ${group.grants.length}`;
          return (
            <div key={group.category}>
              <div className="border-subtle bg-canvasSubtle flex items-center justify-between border-b px-3 py-1.5">
                <span className="text-muted text-[10px] font-semibold uppercase tracking-wider">
                  {group.category}
                </span>
                <span className="text-light font-mono text-[10px]">
                  {count}
                </span>
              </div>
              {group.grants.map((g) => {
                const allowed = permitted.has(g.grant);
                const on = allowed && selectedSet.has(g.grant);
                return (
                  <div
                    key={g.grant}
                    className="border-subtle flex items-center gap-3 border-b px-3 py-1.5 last:border-b-0"
                  >
                    {restricted &&
                      (!allowed ? (
                        <RiLockLine className="text-disabled h-3.5 w-3.5 shrink-0" />
                      ) : on ? (
                        <RiCheckLine className="text-success h-3.5 w-3.5 shrink-0" />
                      ) : (
                        <RiSubtractLine className="text-disabled h-3.5 w-3.5 shrink-0" />
                      ))}
                    <div className="flex min-w-0 flex-1 flex-col">
                      <span
                        className={cn(
                          'font-mono text-xs',
                          allowed ? 'text-basis' : 'text-disabled',
                        )}
                      >
                        {g.grant}
                      </span>
                      <span
                        className={cn(
                          'text-[11px]',
                          allowed ? 'text-light' : 'text-disabled',
                        )}
                      >
                        {allowed
                          ? g.description
                          : 'Disallowed by your account admins'}
                      </span>
                    </div>
                    {allowed ? (
                      <Switch
                        size="sm"
                        className="shrink-0"
                        checked={on}
                        disabled={disabled}
                        aria-label={g.grant}
                        onCheckedChange={(checked) => toggle(g.grant, checked)}
                      />
                    ) : (
                      <span className="text-disabled shrink-0 text-xs">
                        Not allowed
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
