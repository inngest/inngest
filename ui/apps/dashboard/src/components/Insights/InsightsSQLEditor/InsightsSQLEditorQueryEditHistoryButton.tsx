'use client';

import { useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button/Button';
import { Popover, PopoverContent, PopoverTrigger } from '@inngest/components/Popover';
import {
  format as formatDate,
  isValid,
  relativeTime,
  toMaybeDate,
} from '@inngest/components/utils/date';

import { useStoredQueries } from '@/components/Insights/QueryHelperPanel/StoredQueriesContext';
import type { Tab } from '../types';

type InsightsSQLEditorQueryEditHistoryButtonProps = { tab: Tab };

export function InsightsSQLEditorQueryEditHistoryButton({
  tab,
}: InsightsSQLEditorQueryEditHistoryButtonProps) {
  const { queries } = useStoredQueries();
  const [open, setOpen] = useState(false);

  const savedQuery = useMemo(() => {
    if (tab.savedQueryId === undefined) return undefined;

    return queries.data?.find((q) => q.id === tab.savedQueryId);
  }, [queries.data, tab.savedQueryId]);

  if (savedQuery === undefined) return null;

  const { createdAt, creator, lastEditor, updatedAt } = savedQuery;

  const hasEdits = createdAt !== updatedAt;

  const labelRelative = getSafeRelativeText(hasEdits ? updatedAt : createdAt);
  if (labelRelative === undefined) return null;

  return (
    <Popover onOpenChange={setOpen} open={open}>
      <PopoverTrigger asChild>
        <div
          onMouseEnter={() => setOpen(true)}
          onMouseLeave={() => setOpen(false)}
          onPointerDown={(e) => {
            e.preventDefault();
          }}
          onClick={(e) => {
            e.preventDefault();
          }}
        >
          <Button
            appearance="ghost"
            className="active:bg-canvasSubtle focus:bg-canvasSubtle text-muted font-medium"
            kind="secondary"
            label={`${hasEdits ? 'Edited' : 'Created'} ${labelRelative}`}
            size="medium"
          />
        </div>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="text-subtle flex flex-col gap-2 rounded-lg p-3 text-sm leading-tight shadow-[0_1px_1px_0_rgba(0,0,0,0.06),_0_1px_2px_0_rgba(0,0,0,0.10)]"
        onOpenAutoFocus={(e) => e.preventDefault()}
        side="bottom"
      >
        <AuthorshipDate author={creator} date={createdAt} label="Created by" />
        {hasEdits && <AuthorshipDate author={lastEditor} date={updatedAt} label="Edited by" />}
      </PopoverContent>
    </Popover>
  );
}

type AuthorshipDateProps = { author: string; date: string; label: string };

function AuthorshipDate({ author, date, label }: AuthorshipDateProps) {
  const d = toMaybeDate(date);

  return (
    <div>
      <span className="text-muted">{label}</span>{' '}
      <span className="text-basis font-medium">{author}</span>{' '}
      {d !== null && isValid(d) && (
        <time className="text-muted" dateTime={d.toISOString()}>
          {formatDate(d, 'dd MMM, yyyy')}
        </time>
      )}
    </div>
  );
}

function getSafeRelativeText(value: string): undefined | string {
  const d = toMaybeDate(value);
  if (d === null || !isValid(d)) return undefined;

  return relativeTime(d);
}
