import { useEffect, useState } from 'react';
import { RiSearchLine } from '@remixicon/react';

import { QuickSearchModal } from '../Navigation/QuickSearch/QuickSearchModal';

export default function SearchTrigger({
  envSlug,
  envName,
}: {
  envSlug: string;
  envName: string;
}) {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setIsOpen((open) => !open);
      }
    }

    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, []);

  return (
    <>
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className="bg-canvasMuted border-subtle text-muted hover:bg-canvasMuted/80 flex h-8 w-96 shrink-0 items-center gap-2 whitespace-nowrap rounded border px-2.5 text-sm"
      >
        <RiSearchLine className="h-4 w-4 shrink-0" />
        <span className="flex-1 text-left">Search by name or IDs</span>
        <kbd className="border-subtle text-disabled shrink-0 rounded border px-1.5 py-0.5 text-xs">
          ⌘K
        </kbd>
      </button>
      <QuickSearchModal
        envSlug={envSlug}
        envName={envName}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
      />
    </>
  );
}
