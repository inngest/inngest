import { Checkbox } from '@inngest/components/Checkbox/Checkbox';
import { Button } from '@inngest/components/Button';

import {
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
   * Grants this caller may select. Anything outside it renders disabled with an
   * explanation, rather than being hidden — a member should be able to see that
   * a permission exists and that an admin controls it.
   */
  permitted: Set<string>;
  /** Shown once above the list when some grants are unavailable. */
  restrictionNote?: string;
  disabled?: boolean;
};

export function GrantPicker({
  grants,
  selected,
  onChange,
  permitted,
  restrictionNote,
  disabled = false,
}: Props) {
  const groups = groupGrants(grants);
  const selectedSet = new Set(selected);

  function toggle(grant: string, checked: boolean) {
    const next = new Set(selectedSet);
    if (checked) {
      next.add(grant);
    } else {
      next.delete(grant);
    }
    onChange([...next].sort());
  }

  function applyPreset(preset: string[]) {
    // Presets never select something the caller may not mint, so a member
    // clicking Full Access gets the most they are allowed rather than an error.
    onChange(preset.filter((g) => permitted.has(g)).sort());
  }

  const hasRestrictions = grants.some((g) => !permitted.has(g.grant));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-basis text-sm font-medium">Permissions</span>
        <div className="flex gap-2">
          <Button
            appearance="outlined"
            size="small"
            kind="secondary"
            disabled={disabled}
            onClick={() => applyPreset(readOnlyPreset(grants))}
            label="Read only"
          />
          <Button
            appearance="outlined"
            size="small"
            kind="secondary"
            disabled={disabled}
            onClick={() => applyPreset(fullAccessPreset(grants))}
            label="Full access"
          />
        </div>
      </div>

      {hasRestrictions && restrictionNote && (
        <p className="text-muted text-xs">{restrictionNote}</p>
      )}

      <div className="border-subtle divide-subtle max-h-80 divide-y overflow-y-auto rounded border">
        {groups.map((group) => (
          <div key={group.category} className="p-3">
            <p className="text-muted mb-2 text-xs font-medium uppercase tracking-wide">
              {group.category}
            </p>
            <div className="flex flex-col gap-2">
              {group.resources.map((resource) => (
                <div
                  key={resource.name}
                  className="flex items-start justify-between gap-4"
                >
                  <div className="min-w-0">
                    <p className="text-basis text-sm">{resource.name}</p>
                    <p className="text-muted text-xs">{resource.description}</p>
                  </div>
                  <div className="flex shrink-0 gap-4">
                    {(['read', 'write'] as const).map((action) => {
                      const grant = resource[action];
                      if (!grant) {
                        // This resource has no endpoint for that action, so
                        // there is nothing to grant.
                        return <span key={action} className="w-14" />;
                      }
                      const allowed = permitted.has(grant);
                      return (
                        <label
                          key={action}
                          className="flex w-14 items-center gap-1.5"
                          title={
                            allowed
                              ? grant
                              : `${grant} — not available on this account. Ask an org admin.`
                          }
                        >
                          <Checkbox
                            checked={selectedSet.has(grant)}
                            disabled={disabled || !allowed}
                            onCheckedChange={(checked) =>
                              toggle(grant, checked === true)
                            }
                          />
                          <span
                            className={
                              allowed
                                ? 'text-muted text-xs'
                                : 'text-disabled text-xs'
                            }
                          >
                            {action}
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {selected.length === 0 && (
        <p className="text-error text-xs">Select at least one permission.</p>
      )}
    </div>
  );
}
