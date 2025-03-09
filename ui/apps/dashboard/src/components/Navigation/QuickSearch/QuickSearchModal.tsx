import { useState } from 'react';
import { Modal } from '@inngest/components/Modal';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { RiSearchLine } from '@remixicon/react';
import { Command } from 'cmdk';

import { pathCreator } from '@/utils/urls';
import { ResultItem } from './ResultItem';
import Shortcuts from './Shortcuts';
import { useQuickSearch } from './data';
import { useDebounce } from './hooks';

type Props = {
  envSlug: string;
  isOpen: boolean;
  onClose: () => unknown;
};

export function QuickSearchModal({ envSlug, isOpen, onClose }: Props) {
  const [term, setTerm] = useState('');
  const debouncedTerm = useDebounce(term, 1000);
  const isTyping = term !== debouncedTerm;
  const hasSearchTerm = debouncedTerm !== '';

  const res = useQuickSearch({ envSlug, term: debouncedTerm });

  return (
    <Modal alignTop isOpen={isOpen} onClose={onClose} className="max-w-2xl align-baseline">
      <Command label="Type a command or search" shouldFilter={true}>
        <Command.Input
          placeholder="Type a command or search..."
          value={term}
          onValueChange={setTerm}
          className={cn(
            'border-subtle focus:border-subtle placeholder-disabled bg-canvasBase w-[656px] border-x-0 border-b border-t-0 px-4 py-3 outline-none focus:ring-0'
          )}
        />
        <Command.List className="text-subtle bg-canvasBase h-[min(330px,calc(var(--cmdk-list-height)+24px))] overflow-scroll px-4 py-3">
          {(isTyping || res.isFetching) && (
            <Command.Loading className="text-muted text-xs">
              Searching for results matching &quot;{term}&quot;...
              <Skeleton className="mt-1 h-10 w-full" />
            </Command.Loading>
          )}
          {!isTyping && !res.isFetching && hasSearchTerm && (
            <div className="text-muted mb-1 text-xs">
              Search results found for &quot;{debouncedTerm}&quot;
            </div>
          )}

          {!isTyping && !res.isFetching && res.data && !res.error && (
            <Command.Group>
              {res.data.apps.map((app, i) => {
                return (
                  <ResultItem
                    key={app.name}
                    kind="app"
                    onClick={onClose}
                    path={pathCreator.app({ envSlug, externalAppID: app.name })}
                    text={app.name}
                    value={`app-${i}-${app.name}`}
                  />
                );
              })}

              {res.data.event && (
                <ResultItem
                  kind="event"
                  onClick={onClose}
                  path={pathCreator.event({
                    envSlug,
                    eventName: res.data.event.name,
                    eventID: res.data.event.id,
                  })}
                  text={res.data.event.name}
                  value={`event-${res.data.event.name}`}
                />
              )}

              {res.data.eventTypes.map((eventType, i) => {
                return (
                  <ResultItem
                    key={eventType.name}
                    kind="eventType"
                    onClick={onClose}
                    path={pathCreator.eventType({ envSlug, eventName: eventType.name })}
                    text={eventType.name}
                    value={`eventType-${i}-${eventType.name}`}
                  />
                );
              })}

              {res.data.functions.map((fn, i) => {
                return (
                  <ResultItem
                    key={fn.name}
                    kind="function"
                    onClick={onClose}
                    path={pathCreator.function({ envSlug, functionSlug: fn.slug })}
                    text={fn.name}
                    value={`function-${i}-${fn.name}`}
                  />
                );
              })}

              {res.data.run && (
                <ResultItem
                  key={res.data.run.id}
                  kind="run"
                  onClick={onClose}
                  path={pathCreator.runPopout({ envSlug, runID: res.data.run.id })}
                  text={res.data.run.id}
                  value={`run-${res.data.run.id}`}
                />
              )}
            </Command.Group>
          )}
          {!isTyping && !res.isFetching && (
            <Shortcuts onClose={onClose} envSlug={envSlug} hasSearchTerm={hasSearchTerm} />
          )}

          <Command.Empty
            className={cn(
              'text-muted flex h-10 items-center gap-2 px-2 text-sm',
              !res.error && 'hidden'
            )}
          >
            <RiSearchLine className="text-light h-4 w-4" />
            Error searching
          </Command.Empty>

          <Command.Empty
            className={cn(
              'text-muted flex h-10 items-center gap-2 px-2 text-sm',
              (isTyping || res.isPending || res.error) && 'hidden'
            )}
          >
            <RiSearchLine className="text-light h-4 w-4" />
            No results found for <span className="text-basis">&quot;{debouncedTerm}&quot;</span>
          </Command.Empty>
        </Command.List>
      </Command>
    </Modal>
  );
}
