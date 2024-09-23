'use client';

import { useEffect, useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { useOrganization, useUser } from '@clerk/nextjs';
import { NewButton } from '@inngest/components/Button';
import { Modal } from '@inngest/components/Modal';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowGoForwardLine } from '@remixicon/react';
import { Command } from 'cmdk';
import { useQuery } from 'urql';

import { getBaseRunUrl } from '@/components/Runs/utils';
import { graphql } from '@/gql';
import { getEnvironmentSlug } from '@/utils/environments';

const GetGlobalSearchDocument = graphql(`
  query GetGlobalSearch($opts: SearchInput!) {
    account {
      search(opts: $opts) {
        results {
          env {
            name
            id
            type
            slug
          }
          kind
          value {
            ... on ArchivedEvent {
              id
              name
            }
            ... on FunctionRun {
              id
              functionID: workflowID
              startedAt
            }
          }
        }
      }
    }
  }
`);

const GetFunctionSlugDocument = graphql(`
  query GetFunctionSlug($environmentID: ID!, $functionID: ID!) {
    environment: workspace(id: $environmentID) {
      function: workflow(id: $functionID) {
        slug
        name
      }
    }
  }
`);

type SearchModalProps = {
  isOpen: boolean;
  onOpenChange(open: boolean): void;
};

function SearchModal({ isOpen, onOpenChange }: SearchModalProps) {
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [isDebouncing, setIsDebouncing] = useState(false);
  const router = useRouter();
  const { user } = useUser();
  const { organization } = useOrganization();
  let searchResult = {
    type: '',
    href: '',
    name: '',
    icon: <></>,
  };

  useEffect(() => {
    const debounce = setTimeout(() => {
      setIsDebouncing(false);
      setDebouncedSearch(search);
    }, 1000);

    setIsDebouncing(true);

    return () => {
      clearTimeout(debounce);
    };
  }, [search]);

  useEffect(() => {
    if (debouncedSearch && user && organization) {
      window.inngest.send({
        name: 'app/global.id.searched',
        data: {
          word: search,
          result: {
            type: searchResult.type,
            name: searchResult.name,
          },
        },
        user: {
          external_id: user.externalId,
          email: user.primaryEmailAddress?.emailAddress,
          name: user.fullName,
          account_id: organization.publicMetadata.accountID,
        },
        v: '2024-02-06.1',
      });
    }
  }, [debouncedSearch, user]);

  /*
   * Collects the search data based on either the FunctionRunID or EventID
   */
  const [{ data: globalSearchData, fetching: globalSearchFetching }] = useQuery({
    query: GetGlobalSearchDocument,
    variables: {
      opts: {
        term: debouncedSearch,
      },
    },
    pause: !debouncedSearch,
  });
  const globalResults = globalSearchData?.account.search.results[0];

  /*
   * Collects the function slug when the search matched a FunctionRunID
   */
  const [{ data: getFunctionSlugData, fetching: getFunctionSlugFetching }] = useQuery({
    query: GetFunctionSlugDocument,
    variables: {
      environmentID: globalResults?.env.id || '',
      functionID:
        globalResults?.value.__typename === 'FunctionRun' ? globalResults.value.functionID : '',
    },
    pause: !globalSearchData || globalResults?.value.__typename === 'ArchivedEvent',
  });
  const functionResults = getFunctionSlugData?.environment.function;
  const isFetching = globalSearchFetching || getFunctionSlugFetching;

  /*
   * Returns the environment slug
   */
  const environmentSlug = getEnvironmentSlug({
    environmentID: globalResults?.env.id ?? '',
    environmentSlug: globalResults?.env.slug || null,
    environmentName: globalResults?.env.name ?? '',
    environmentType: globalResults?.env.type ?? '',
  });

  /*
   * Generates the result to be displayed to the user
   */
  if (globalResults?.value.__typename === 'FunctionRun' && functionResults) {
    // runs from before clickhouse migration go to the old runs page
    const baseUrl = getBaseRunUrl(globalResults.value.startedAt);
    searchResult = {
      type: 'function',
      href: `/env/${environmentSlug}/functions/${encodeURIComponent(
        functionResults.slug
      )}/${baseUrl}/${globalResults.value.id}`,
      name: functionResults.name || '',
      icon: <IconFunction className="w-4" />,
    };
  } else if (globalResults?.value.__typename === 'ArchivedEvent') {
    searchResult = {
      type: 'event',
      href: `/env/${environmentSlug}/events/${encodeURIComponent(globalResults.value.name)}/logs/${
        globalResults.value.id
      }`,
      name: globalResults.value.name,
      icon: <IconEvent className="w-5" />,
    };
  }

  const isLoading = isFetching || isDebouncing;

  return (
    <Modal alignTop isOpen={isOpen} onClose={onOpenChange} className="max-w-2xl align-baseline">
      <Command label="Search by ID menu" shouldFilter={false} className="p-2">
        <Command.Input
          placeholder="Search by ID..."
          value={search}
          onValueChange={setSearch}
          className={cn(
            search && 'border-b border-slate-200 focus:border-slate-200',
            'w-[656px] border-0 px-3 py-3 placeholder-slate-500 outline-none focus:ring-0'
          )}
        />
        {search && (
          <Command.List className="px-3 py-3 text-slate-600">
            {isLoading && <Command.Loading>Searching...</Command.Loading>}
            {!isLoading && globalResults && (
              <Command.Group
                heading={<div className="pb-2 text-xs text-slate-500">Navigate To</div>}
              >
                <Command.Item
                  onSelect={() => {
                    router.push(searchResult.href as Route);
                    onOpenChange(!isOpen);
                  }}
                  key={globalResults.env.id}
                  value={globalResults.env.name}
                  className="group flex cursor-pointer items-center rounded-md px-3 py-3 data-[selected]:bg-slate-100"
                >
                  <div className="flex items-center gap-2 truncate">
                    {searchResult.icon}
                    <p className="flex-1 truncate">{searchResult.name}</p>
                  </div>
                  <kbd
                    aria-label="press enter to jump to page"
                    className="ml-auto hidden rounded bg-slate-200 p-1.5 text-white group-data-[selected]:block"
                  >
                    <RiArrowGoForwardLine className="h-3 w-3 rotate-180 text-slate-600" />
                  </kbd>
                </Command.Item>
              </Command.Group>
            )}
            <Command.Empty className={cn(isLoading && 'hidden')}>
              No results found. Make sure you are typing the full ID.
            </Command.Empty>
          </Command.List>
        )}
      </Command>
    </Modal>
  );
}

export default function Search({ collapsed }: { collapsed: boolean }) {
  const [isSearchModalVisible, setIsSearchModalVisible] = useState(false);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setIsSearchModalVisible((open) => !open);
      }
    }

    document.addEventListener('keydown', onKeyDown);

    return () => {
      document.removeEventListener('keydown', onKeyDown);
    };
  }, []);

  return (
    <>
      {collapsed ? null : (
        <Tooltip>
          <TooltipTrigger asChild>
            <NewButton
              kind="secondary"
              appearance="outlined"
              size="medium"
              className="h-[28px] w-[42px] overflow-hidden px-2"
              onClick={() => setIsSearchModalVisible(true)}
              aria-label="Search by ID"
              icon={
                <kbd className="mx-auto flex w-full items-center justify-center space-x-1">
                  <kbd className={`text-muted text-[20px]`}>⌘</kbd>
                  <kbd className="text-muted text-xs">K</kbd>
                </kbd>
              }
            />
          </TooltipTrigger>
          <TooltipContent
            side="bottom"
            sideOffset={2}
            className="border-muted text-muted rounded border text-xs"
          >
            Use <span className="font-bold">⌘ K</span> or <span className="font-bold">Ctrl K</span>{' '}
            to search
          </TooltipContent>
        </Tooltip>
      )}

      <SearchModal isOpen={isSearchModalVisible} onOpenChange={setIsSearchModalVisible} />
    </>
  );
}
