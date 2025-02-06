import { useState } from 'react';
import { Modal } from '@inngest/components/Modal';
import { cn } from '@inngest/components/utils/classNames';
import { Command } from 'cmdk';

import { pathCreator } from '@/utils/urls';
import { ResultItem } from './ResultItem';
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

  const res = useQuickSearch({ envSlug, term: debouncedTerm });

  return (
    <Modal alignTop isOpen={isOpen} onClose={onClose} className="max-w-2xl align-baseline">
      <Command label="Search by ID menu" shouldFilter={false} className="p-2">
        <Command.Input
          placeholder="Search by ID or name..."
          value={term}
          onValueChange={setTerm}
          className={cn(
            debouncedTerm && 'border-subtle focus:border-subtle border-b',
            'placeholder-disabled bg-canvasBase w-[656px] border-0 px-3 py-3 outline-none focus:ring-0'
          )}
        />
        {debouncedTerm && (
          <Command.List className="text-subtle bg-canvasBase px-3 py-3">
            {(isTyping || res.isFetching) && <Command.Loading>Searching...</Command.Loading>}

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

            <Command.Empty className={cn(!res.error && 'hidden')}>Error searching</Command.Empty>

            <Command.Empty className={cn((isTyping || res.isPending || res.error) && 'hidden')}>
              No results found
            </Command.Empty>
          </Command.List>
        )}
      </Command>
    </Modal>
  );
}
