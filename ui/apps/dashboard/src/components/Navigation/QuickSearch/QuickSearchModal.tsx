import { useState } from 'react';
import { Modal } from '@inngest/components/Modal';
import { Pill } from '@inngest/components/Pill';
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
  envName: string;
  isOpen: boolean;
  onClose: () => unknown;
};

export function QuickSearchModal({ envSlug, envName, isOpen, onClose }: Props) {
  const [term, setTerm] = useState('');
  const debouncedTerm = useDebounce(term, 200);
  const isTyping = term !== debouncedTerm;

  const res = useQuickSearch({ envSlug, term: debouncedTerm });

  return (
    <Modal
      alignTop
      isOpen={isOpen}
      onClose={onClose}
      className="mt-cmdk-margin max-w-2xl"
    >
      <Command
        label="Search by functions, events, apps and IDs"
        shouldFilter={true}
      >
        <div className="border-subtle bg-modalBase border-b px-4 py-3">
          <Pill appearance="solidBright" className="mb-3">
            {envName}
          </Pill>
          <Command.Input
            placeholder="Search by functions, events, apps and IDs"
            value={term}
            onValueChange={setTerm}
            className={cn(
              'placeholder-disabled bg-modalBase w-[656px] border-0 p-0 outline-none focus:ring-0',
            )}
          />
        </div>
        <Command.List className="text-subtle bg-modalBase h-[min(330px,calc(var(--cmdk-list-height)+24px))] overflow-auto px-4 py-3">
          {(isTyping || res.isFetching) && (
            <Command.Loading className="text-light text-xs">
              <div className="flex items-center gap-2 px-2">
                <RiSearchLine className="text-light h-4 w-4" />
                Searching for results matching &quot;{term}&quot;...
              </div>
              <Skeleton className="mt-1 h-10 w-full" />
            </Command.Loading>
          )}

          {!isTyping && !res.isFetching && res.data && !res.error && (
            <>
              {res.data.run && (
                <Command.Group
                  heading="Runs"
                  className="text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1"
                >
                  <ResultItem
                    isDifferentEnv={
                      envSlug !== coerceEnvSlug(res.data.run.envSlug)
                    }
                    key={res.data.run.id}
                    kind="run"
                    onClick={onClose}
                    path={pathCreator.runPopout({
                      envSlug: coerceEnvSlug(res.data.run.envSlug),
                      runID: res.data.run.id,
                    })}
                    text={res.data.run.id}
                    value={`run-${res.data.run.id}`}
                  />
                </Command.Group>
              )}
              <Command.Group
                heading="Apps"
                className="text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1"
              >
                {res.data.apps.map((app, i) => {
                  return (
                    <ResultItem
                      key={app.name}
                      kind="app"
                      onClick={onClose}
                      path={pathCreator.app({
                        envSlug,
                        externalAppID: app.name,
                      })}
                      text={app.name}
                      value={`app-${i}-${app.name}`}
                    />
                  );
                })}
              </Command.Group>
              <Command.Group
                heading="Functions"
                className="text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1"
              >
                {res.data.functions.map((fn, i) => {
                  return (
                    <ResultItem
                      key={fn.name}
                      kind="function"
                      onClick={onClose}
                      path={pathCreator.function({
                        envSlug,
                        functionSlug: fn.slug,
                      })}
                      text={fn.name}
                      value={`function-${i}-${fn.name}`}
                    />
                  );
                })}
              </Command.Group>
              {res.data.event && (
                <Command.Group
                  heading="Events"
                  className="text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1"
                >
                  <ResultItem
                    isDifferentEnv={
                      envSlug !== coerceEnvSlug(res.data.event.envSlug)
                    }
                    key={res.data.event.id}
                    kind="event"
                    onClick={onClose}
                    path={pathCreator.eventPopout({
                      envSlug: coerceEnvSlug(res.data.event.envSlug),
                      eventID: res.data.event.id,
                    })}
                    text={res.data.event.name}
                    value={`event-${res.data.event.id}`}
                  />
                </Command.Group>
              )}
              <Command.Group
                heading="Event Types"
                className="text-muted mb-4 text-xs [&_[cmdk-group-heading]]:mb-1"
              >
                {res.data.eventTypes.map((eventType, i) => {
                  return (
                    <ResultItem
                      key={eventType.name}
                      kind="eventType"
                      onClick={onClose}
                      path={pathCreator.eventType({
                        envSlug,
                        eventName: eventType.name,
                      })}
                      text={eventType.name}
                      value={`eventType-${i}-${eventType.name}`}
                    />
                  );
                })}
              </Command.Group>
            </>
          )}
          {!isTyping && !res.isFetching && (
            <Shortcuts onClose={onClose} envSlug={envSlug} />
          )}

          <Command.Empty
            className={cn(
              'text-muted flex h-10 items-center gap-2 px-2 text-sm',
              !res.error && 'hidden',
            )}
          >
            <RiSearchLine className="text-light h-4 w-4" />
            Error searching
          </Command.Empty>

          <Command.Empty
            className={cn(
              'text-muted flex h-10 items-center gap-2 px-2 text-sm',
              (isTyping || res.isPending || res.error) && 'hidden',
            )}
          >
            <RiSearchLine className="text-light h-4 w-4" />
            No results found for{' '}
            <span className="text-basis">&quot;{debouncedTerm}&quot;</span>
          </Command.Empty>
        </Command.List>
      </Command>
    </Modal>
  );
}

function coerceEnvSlug(envSlug: string): string {
  if (envSlug && envSlug.startsWith('production')) {
    // This is hacky and flawed. The production env has a pseudo slug in the URL
    // ("production") which will never match its real slug in the DB. So we'll
    // coerce the real slug to the pseudo slug.
    //
    // This doesn't work if the user created a non-production env that starts
    // with "production", but should otherwise be fine. This also won't work
    // when we add support for multiple production environments.
    return 'production';
  }

  return envSlug;
}
