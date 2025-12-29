import { Button } from '@inngest/components/Button/NewButton';
import { RiExternalLinkLine } from '@remixicon/react';

import { SHOW_EXAMPLES_LINK } from '@/components/Insights/temp-flags';
import { IconLayoutWrapper } from './IconLayoutWrapper';

export function EmptyState() {
  const subheaderPrefix =
    'Run a query to analyze your data and the results will be displayed here.';
  const subheader = SHOW_EXAMPLES_LINK
    ? `${subheaderPrefix} If you need a starting point, check out our examples.`
    : subheaderPrefix;

  return (
    <IconLayoutWrapper
      action={
        SHOW_EXAMPLES_LINK ? (
          <Button
            appearance="outlined"
            disabled
            icon={<RiExternalLinkLine />}
            iconSide="left"
            kind="primary"
            label="See examples"
          />
        ) : null
      }
      header="Your query results will appear here"
      subheader={subheader}
    />
  );
}
