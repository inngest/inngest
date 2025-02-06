'use client';

import { useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';

import { QuickSearchModal } from './QuickSearchModal';

type Props = {
  collapsed: boolean;
  envSlug: string;
};

export function QuickSearch({ collapsed, envSlug }: Props) {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setIsOpen((open) => !open);
      }
    }

    document.addEventListener('keydown', onKeyDown);

    return () => {
      document.removeEventListener('keydown', onKeyDown);
    };
  }, []);

  if (collapsed) {
    return null;
  }

  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            appearance="outlined"
            aria-label="Search by ID"
            className="h-[28px] w-[42px] overflow-hidden px-2"
            icon={
              <kbd className="mx-auto flex w-full items-center justify-center space-x-1">
                <kbd className={`text-muted text-[20px]`}>⌘</kbd>
                <kbd className="text-muted text-xs">K</kbd>
              </kbd>
            }
            kind="secondary"
            onClick={() => setIsOpen(true)}
            size="medium"
          />
        </TooltipTrigger>

        <TooltipContent
          className="border-muted text-muted rounded border text-xs"
          side="bottom"
          sideOffset={2}
        >
          Use <span className="font-bold">⌘ K</span> or <span className="font-bold">Ctrl K</span> to
          search
        </TooltipContent>
      </Tooltip>

      <QuickSearchModal envSlug={envSlug} isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}
