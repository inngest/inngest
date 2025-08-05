'use client';

import { useQueryHelperPanelContext } from './QueryHelperPanelContext';
import { QueryHelperPanelSection } from './QueryHelperPanelSection';

export function QueryHelperPanel() {
  const { recentQueries, savedQueries, templates } = useQueryHelperPanelContext();

  return (
    <div className="border-subtle flex h-full w-full flex-col border-r">
      <QueryHelperPanelSection queries={templates} title="Templates" />
      <div className="border-subtle border-t" />
      <QueryHelperPanelSection queries={recentQueries} title="Recent queries" />
      <div className="border-subtle border-t" />
      <QueryHelperPanelSection queries={savedQueries} title="Saved queries" />
    </div>
  );
}
