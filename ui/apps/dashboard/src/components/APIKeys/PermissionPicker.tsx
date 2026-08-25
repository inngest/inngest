import { permissionResourceCopy } from './permissionResourceCopy';

export type PermissionGroup = {
  resource: string;
  read: string[];
  write: string[];
};

export type PermissionLevel = 'none' | 'read' | 'write';

type Props = {
  groups: PermissionGroup[];
  levels: Record<string, PermissionLevel>;
  disabled?: boolean;
  onChange: (resource: string, level: PermissionLevel) => void;
};

export function PermissionPicker({
  groups,
  levels,
  disabled = false,
  onChange,
}: Props) {
  const sortedGroups = [...groups].sort((a, b) =>
    a.resource.localeCompare(b.resource),
  );

  return (
    <div className="border-subtle rounded border">
      {sortedGroups.map((group) => {
        const level = levels[group.resource] ?? 'none';
        const copy = permissionResourceCopy(group.resource);

        return (
          <div
            key={group.resource}
            className="border-subtle grid grid-cols-1 gap-3 border-b p-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
          >
            <div className="flex min-w-0 flex-col">
              <span className="text-basis truncate text-sm font-medium">
                {copy.label}
              </span>
              {copy.description && (
                <span className="text-subtle truncate text-xs">
                  {copy.description}
                </span>
              )}
            </div>

            <div className="bg-canvasMuted grid h-8 grid-cols-3 rounded-full p-0.5">
              <PermissionButton
                active={level === 'none'}
                disabled={disabled}
                label="None"
                onClick={() => onChange(group.resource, 'none')}
              />
              <PermissionButton
                active={level === 'read'}
                disabled={disabled || group.read.length === 0}
                label="Read"
                onClick={() => onChange(group.resource, 'read')}
              />
              <PermissionButton
                active={level === 'write'}
                disabled={disabled || group.write.length === 0}
                label="Write"
                onClick={() => onChange(group.resource, 'write')}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function PermissionButton({
  active,
  disabled,
  label,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  label: string;
  onClick: () => void;
}) {
  const classes = [
    'h-7 min-w-16 rounded-full px-3 text-sm outline-none transition-colors',
    'disabled:text-disabled disabled:cursor-not-allowed',
  ];
  if (active) {
    classes.push('bg-canvasBase border-muted text-basis border');
  } else {
    classes.push(
      'text-muted hover:bg-canvasSubtle hover:text-basis border border-transparent',
    );
  }

  return (
    <button
      type="button"
      className={classes.join(' ')}
      onClick={onClick}
      disabled={disabled}
    >
      {label}
    </button>
  );
}
