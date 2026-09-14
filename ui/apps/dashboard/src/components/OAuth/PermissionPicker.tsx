import { useId, type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import { Table } from '@inngest/components/Table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { RiInformationLine } from '@remixicon/react';
import type { ColumnDef } from '@tanstack/react-table';

import { permissionResourceCopy } from './permissionResourceCopy';
import { bulkPermissionLevels } from './permissionSelection';

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
  onChange: (levels: Record<string, PermissionLevel>) => void;
};

const permissionOptions = [
  { level: 'none', label: 'None', shortcut: 'No permissions' },
  { level: 'read', label: 'Read', shortcut: 'Read all' },
  { level: 'write', label: 'Write', shortcut: 'Write all' },
] as const;

type PermissionRow = {
  resource: string;
  choices: Record<string, ReactNode>;
};

// stable cell renderers keep keyboard focus when a selection changes
const columns: ColumnDef<PermissionRow>[] = [
  {
    id: 'resource',
    header: () => <span className="uppercase">Resources</span>,
    cell: ({ row }) => {
      const copy = permissionResourceCopy(row.original.resource);
      return (
        <div className="flex flex-col gap-1 py-2">
          <span className="text-basis text-sm">{copy.label}</span>
          {copy.description && (
            <span className="text-subtle text-xs">{copy.description}</span>
          )}
        </div>
      );
    },
  },
  ...permissionOptions.map(
    ({ level, label }): ColumnDef<PermissionRow> => ({
      id: level,
      header: () => (
        <span className="flex items-center justify-center gap-1 uppercase">
          {label}
          {level === 'write' && (
            <Tooltip>
              <TooltipTrigger asChild>
                <button type="button" aria-label="About write access">
                  <RiInformationLine className="h-3.5 w-3.5" />
                </button>
              </TooltipTrigger>
              <TooltipContent>Write includes read access.</TooltipContent>
            </Tooltip>
          )}
        </span>
      ),
      cell: ({ row }) => row.original.choices[level],
    }),
  ),
];

export function PermissionPicker({
  groups,
  levels,
  disabled = false,
  onChange,
}: Props) {
  const id = useId();
  const rows: PermissionRow[] = [...groups]
    .sort((a, b) => a.resource.localeCompare(b.resource))
    .map((group) => ({
      resource: group.resource,
      choices: Object.fromEntries(
        permissionOptions.map(({ level, label }) => [
          level,
          <div className="flex justify-center">
            {level === 'none' || group[level].length > 0 ? (
              <input
                type="radio"
                name={`${id}-${group.resource}`}
                aria-label={`${
                  permissionResourceCopy(group.resource).label
                }: ${label}`}
                value={level}
                checked={(levels[group.resource] ?? 'none') === level}
                disabled={disabled}
                onChange={() =>
                  onChange({ ...levels, [group.resource]: level })
                }
                className="border-muted checked:border-primary-moderate checked:bg-primary-moderate focus-visible:outline-primary-moderate h-4 w-4 cursor-pointer appearance-none rounded-full border bg-clip-content p-[3px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            ) : (
              <span className="text-disabled" aria-label="Not available">
                —
              </span>
            )}
          </div>,
        ]),
      ),
    }));

  return (
    <TooltipProvider>
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-2">
          {permissionOptions.map(({ level, shortcut }) => (
            <Button
              key={level}
              kind="secondary"
              appearance="outlined"
              size="small"
              label={shortcut}
              disabled={disabled}
              onClick={() => onChange(bulkPermissionLevels(groups, level))}
            />
          ))}
        </div>
        <div className="border-subtle overflow-x-auto rounded border">
          <Table data={rows} columns={columns} />
        </div>
      </div>
    </TooltipProvider>
  );
}
