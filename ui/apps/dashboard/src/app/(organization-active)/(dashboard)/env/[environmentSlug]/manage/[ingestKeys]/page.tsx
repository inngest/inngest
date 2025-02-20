'use client';

import useManagePageTerminology from './useManagePageTerminology';

export default function EventKeysPage() {
  const currentContent = useManagePageTerminology();

  return (
    <div className="flex h-full w-full items-center justify-center">
      <h2 className="text-subtle text-sm font-semibold">
        {'Select a ' + currentContent?.type + ' on the left.'}
      </h2>
    </div>
  );
}
