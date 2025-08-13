'use client';

import { Button } from '@inngest/components/Button';
import { RiExternalLinkLine } from '@remixicon/react';

import { IconLayoutWrapper } from './IconLayoutWrapper';

// TODO: Add link to examples.
export function EmptyState() {
  return (
    <IconLayoutWrapper
      action={
        <Button
          appearance="outlined"
          disabled
          icon={<RiExternalLinkLine />}
          iconSide="left"
          kind="primary"
          label="See examples"
        />
      }
      header="Your query results will appear here"
      subheader="Run a query to analyze your data and the results will be displayed here. If you need a starting point, check out our examples."
    />
  );
}
