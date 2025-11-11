'use client';

import { useEffect, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';
import { Button } from '@inngest/components/Button/Button';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiBookmarkLine, RiCloseLargeLine, RiHistoryLine } from '@remixicon/react';

import type { QuerySnapshot } from '@/components/Insights/types';
import type { InsightsQueryStatement } from '@/gql/graphql';

interface QueryHelperPanelSectionItemProps {
  activeSavedQueryId?: string;
  onQueryDelete: (queryId: string) => void;
  onQuerySelect: (query: InsightsQueryStatement | QuerySnapshot) => void;
  query: InsightsQueryStatement | QuerySnapshot;
  sectionType: 'history' | 'saved' | 'shared';
}

export function QueryHelperPanelSectionItem({
  activeSavedQueryId,
  onQueryDelete,
  onQuerySelect,
  query,
  sectionType,
}: QueryHelperPanelSectionItemProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const displayText = query.name;
  const Icon = sectionType === 'history' ? RiHistoryLine : RiBookmarkLine;

  const isActiveTab =
    (sectionType === 'saved' || sectionType === 'shared') && activeSavedQueryId === query.id;

  useEffect(() => {
    const el = textRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [displayText]);

  return (
    <>
      <OptionalTooltip side="right" tooltip={isTruncated ? displayText : ''}>
        <div
          className={cn(
            'text-subtle flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors',
            isActiveTab ? 'bg-canvasSubtle' : 'hover:bg-canvasSubtle'
          )}
          onClick={() => {
            onQuerySelect(query);
          }}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
        >
          <Icon className="h-4 w-4 flex-shrink-0" />
          <span
            ref={textRef}
            className="flex-1 overflow-hidden truncate text-ellipsis whitespace-nowrap"
          >
            {displayText}
          </span>
          <div className="flex h-4 w-4 flex-shrink-0 items-center justify-center">
            <Button
              appearance="ghost"
              className={cn(
                'text-subtle h-4 w-4 p-0 transition-all',
                isHovered ? 'opacity-100' : 'opacity-0'
              )}
              icon={<RiCloseLargeLine className="h-3 w-3" />}
              onClick={(e: React.MouseEvent) => {
                e.stopPropagation();

                if (sectionType === 'saved' || sectionType === 'shared') setShowDeleteModal(true);
                else onQueryDelete(query.id);
              }}
              size="small"
              tooltip="Delete query"
            />
          </div>
        </div>
      </OptionalTooltip>

      <AlertModal
        cancelButtonLabel="Cancel"
        className="w-[656px]"
        confirmButtonLabel="Remove"
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        onSubmit={() => {
          onQueryDelete(query.id);
          setShowDeleteModal(false);
        }}
        title="Remove query"
      >
        <div className="p-6">
          <p className="text-subtle text-sm">
            Are you sure you want to delete <strong>{query.name}</strong> permanently?
          </p>
          <Alert className="mt-4 text-sm" severity="warning">
            This action is permanent and cannot be undone.
          </Alert>
        </div>
      </AlertModal>
    </>
  );
}
