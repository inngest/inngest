import Tabs from '@inngest/components/Tabs/Tabs';
import { RiAddLine, RiCodeSSlashLine } from '@remixicon/react';

import type { InsightsTabData } from './InsightsPage';

// A trimmed-down version of the dashboard Insights feature's
// InsightsTabManager/InsightsTabsList.tsx -- no context menu, unsaved-changes
// modal, or home/templates tabs, since this basic editor has no save/load.
export function InsightsTabsList({
  activeTabId,
  tabs,
  onValueChange,
  onClose,
  onCreateTab,
}: {
  activeTabId: string;
  tabs: InsightsTabData[];
  onValueChange: (id: string) => void;
  onClose: (id: string) => void;
  onCreateTab: () => void;
}) {
  return (
    <Tabs onClose={onClose} onValueChange={onValueChange} value={activeTabId}>
      <Tabs.List>
        <div className="-mr-px flex overflow-x-auto">
          {tabs.map((tab) => (
            <Tabs.Tab
              key={tab.id}
              iconBefore={<RiCodeSSlashLine size={16} />}
              title={tab.name}
              value={tab.id}
              disallowClose={tabs.length === 1}
            />
          ))}
        </div>
        <div className="border-subtle border-l">
          <Tabs.IconTab icon={<RiAddLine size={16} />} onClick={onCreateTab} />
        </div>
      </Tabs.List>
    </Tabs>
  );
}
