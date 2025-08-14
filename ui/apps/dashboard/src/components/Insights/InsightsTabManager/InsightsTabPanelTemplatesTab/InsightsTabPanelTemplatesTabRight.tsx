import { Button } from '@inngest/components/Button/Button';
import { RiChatPollLine, RiExternalLinkLine, RiQuillPenLine } from '@remixicon/react';

import { useTabManagerActions } from '../TabManagerContext';

const BASE_DOCS_URL = 'https://docs.inngest.com/';

// TODO: Update these to point to the correct URLs.
const RESOURCES = [
  { href: BASE_DOCS_URL, label: 'How to write your own query', icon: RiQuillPenLine },
  { href: BASE_DOCS_URL, label: 'Insights documentation', icon: RiExternalLinkLine },
  { href: BASE_DOCS_URL, label: 'Send us feedback', icon: RiChatPollLine },
];

export function InsightsTabPanelTemplatesTabRight() {
  const { tabManagerActions } = useTabManagerActions();
  return (
    <div className="flex w-[360px] flex-shrink-0 flex-col">
      <div className="border-subtle flex flex-col gap-3 border-b">
        <p className="text-muted mb-1 text-sm">Prefer to write a query yourself?</p>
        <Button
          appearance="outlined"
          className="mb-6 w-fit font-medium"
          kind="secondary"
          label="Start from scratch"
          onClick={tabManagerActions.createNewTab}
        />
      </div>
      <div className="mt-6 flex flex-col gap-4">
        <h3 className="text-muted text-xs font-medium">RESOURCES</h3>
        <div className="flex flex-col gap-3">
          {RESOURCES.map((resource) => {
            const IconComponent = resource.icon;
            return (
              <div className="flex items-center gap-2" key={resource.label}>
                <IconComponent className="text-muted-foreground h-4 w-4" />
                <a
                  className="hover:text-foreground text-muted-foreground text-sm transition-colors hover:underline"
                  href={resource.href}
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  {resource.label}
                </a>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
