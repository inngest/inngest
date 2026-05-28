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
        className="bg-canvasBase border-subtle text-muted hover:bg-canvasBase/90 flex h-8 w-72 items-center gap-2 rounded border px-2.5 text-sm"
      >
        <RiSearchLine className="h-4 w-4" />
        <span className="flex-1 text-left">Search by name or IDs</span>
        <kbd className="border-subtle text-disabled rounded border px-1.5 py-0.5 text-xs">
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
