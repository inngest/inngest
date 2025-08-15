'use client';

import { Button } from '@inngest/components/Button/Button';
import { RiBookmarkLine } from '@remixicon/react';

export function InsightsSQLEditorSaveQueryButton() {
  return (
    <Button
      appearance="outlined"
      icon={<RiBookmarkLine className="h-4 w-4" />}
      kind="secondary"
      onClick={() => {
        // TODO: Implement save query functionality
        console.log('Save query clicked');
      }}
      size="medium"
      title="Save query"
    />
  );
}
