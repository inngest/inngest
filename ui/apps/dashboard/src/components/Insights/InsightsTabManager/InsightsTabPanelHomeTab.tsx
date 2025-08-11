'use client';

import { ulid } from 'ulid';

import { useTemplates } from '@/components/Insights/QueryHelperPanel/mock';
import type { TabManagerActions } from './InsightsTabManager';

interface InsightsTabPanelHomeTabProps {
  tabManagerActions: TabManagerActions;
}

export function InsightsTabPanelHomeTab({ tabManagerActions }: InsightsTabPanelHomeTabProps) {
  const templates = useTemplates();

  return (
    <div className="col-span-1 row-span-2 flex h-full w-full flex-col bg-gray-50 p-8 dark:bg-gray-900">
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Templates</h1>
        <p className="text-muted text-lg">Get started with pre-built query templates</p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {templates.data?.map((template) => (
          <button
            key={template.id}
            className="hover:bg-canvasSubtle border-subtle flex flex-col items-start rounded-lg border bg-white p-6 text-left transition-colors dark:bg-gray-800"
            onClick={() => {
              tabManagerActions.createTab({
                ...template,
                id: ulid(),
                name: 'Untitled query',
              });
            }}
          >
            <h3 className="mb-2 font-semibold">{template.name}</h3>
            <p className="text-subtle text-sm">{template.query.slice(0, 100)}...</p>
          </button>
        ))}
      </div>
    </div>
  );
}
