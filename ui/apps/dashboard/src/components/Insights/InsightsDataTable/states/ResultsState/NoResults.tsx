'use client';

import { Button } from '@inngest/components/Button';
import { RiExternalLinkLine } from '@remixicon/react';

import { IconLayoutWrapper } from '../IconLayoutWrapper';

// TODO: Add link to examples.
export function NoResults() {
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
          size="medium"
        />
      }
      header="No results found"
      subheader="We couldn't find any results matching your search. Try adjusting your query or browse our examples for inspiration."
    />
  );
}
