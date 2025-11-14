'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';
import { AlertModal } from '@inngest/components/Modal/AlertModal';
import { RiCodeBlock, RiHistoryLine, RiSaveLine } from '@remixicon/react';

import type { QuerySnapshot } from '@/components/Insights/types';
import type { InsightsQueryStatement } from '@/gql/graphql';
import { QueryActionsMenu } from '../QueryActionsMenu';
import { isQuerySnapshot } from '../queries';
import { QueryHelperPanelSectionItemRow } from './QueryHelperPanelSectionItemRow';

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
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const displayText = query.name;
  const Icon =
    sectionType === 'history' ? RiHistoryLine : sectionType === 'saved' ? RiSaveLine : RiCodeBlock;

  const isActiveTab =
    (sectionType === 'saved' || sectionType === 'shared') && activeSavedQueryId === query.id;

  return (
    <>
      <QueryActionsMenu
        onOpenChange={setMenuOpen}
        onSelectDelete={() => {
          if (isQuerySnapshot(query)) onQueryDelete(query.id);
          else setShowDeleteModal(true);
        }}
        open={menuOpen}
        query={query}
        trigger={
          <QueryHelperPanelSectionItemRow
            icon={<Icon className="h-4 w-4 flex-shrink-0" />}
            isActive={isActiveTab}
            onClick={(e) => {
              e.preventDefault();
              onQuerySelect(query);
            }}
            onContextMenu={(e) => {
              e.preventDefault();
              setMenuOpen(true);
            }}
            text={displayText}
          />
        }
      />

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
