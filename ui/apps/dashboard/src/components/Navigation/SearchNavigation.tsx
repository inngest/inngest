'use client';

import { useEffect, useState } from 'react';
import { type Route } from 'next';
import { useRouter } from 'next/navigation';
import { useUser } from '@clerk/nextjs';
import {
  ArrowUturnRightIcon,
  CodeBracketSquareIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/20/solid';
import { Modal } from '@inngest/components/Modal';
import { classNames } from '@inngest/components/utils/classNames';
import { Command } from 'cmdk';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import EventIcon from '@/icons/event.svg';
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
  const router = useRouter();
  const { user } = useUser();
  let searchResult = {
    type: '',
    href: '',
    name: '',
    icon: <></>,
  };

  useEffect(() => {
    let debounce = setTimeout(() => {
      if (search && user) {
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
            account_id: user.publicMetadata.accountID,
          },
          v: '2023-05-17.1',
        });
      }
    }, 1000);
    return () => {
      clearTimeout(debounce);
    };
  }, [search, searchResult, user]);

  /*
   * Collects the search data based on either the FunctionRunID or EventID
   */
  const [{ data: globalSearchData, fetching: globalSearchFetching }] = useQuery({
    query: GetGlobalSearchDocument,
    variables: {
      opts: {
        term: search,
      },
    },
    pause: !search,
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
    environmentName: globalResults?.env.name ?? '',
    environmentType: globalResults?.env.type ?? '',
  });

  /*
   * Generates the result to be displayed to the user
   */
  if (globalResults?.value.__typename === 'FunctionRun' && functionResults) {
    searchResult = {
      type: 'function',
      href: `/env/${environmentSlug}/functions/${encodeURIComponent(functionResults.slug)}/logs/${
        globalResults.value.id
      }`,
      name: functionResults.name || '',
      icon: <CodeBracketSquareIcon className="w-4" />,
    };
  } else if (globalResults?.value.__typename === 'ArchivedEvent') {
    searchResult = {
      type: 'event',
      href: `/env/${environmentSlug}/events/${encodeURIComponent(globalResults.value.name)}/logs/${
        globalResults.value.id
      }`,
      name: globalResults.value.name,
      icon: <EventIcon className="w-5" />,
    };
  }

  return (
    <Modal alignTop isOpen={isOpen} onClose={onOpenChange} className="max-w-2xl align-baseline">
      <Command label="Search by ID menu" shouldFilter={false} className="p-2">
        <Command.Input
          placeholder="Search by ID..."
          value={search}
          onValueChange={setSearch}
          className={classNames(
            search && 'border-b border-slate-200 focus:border-slate-200',
            'w-[656px] border-0 px-3 py-3 placeholder-slate-500 outline-none focus:ring-0'
          )}
        />
        {search && (
          <Command.List className="px-3 py-3 text-slate-600">
            {isFetching && <Command.Loading>Searching...</Command.Loading>}
            {!isFetching && globalResults && (
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
                    <ArrowUturnRightIcon className="h-3 w-3 rotate-180 text-slate-600" />
                  </kbd>
                </Command.Item>
              </Command.Group>
            )}
            <Command.Empty className={classNames(isFetching && 'hidden')}>
              No results found. Make sure you are typing the full ID.
            </Command.Empty>
          </Command.List>
        )}
      </Command>
    </Modal>
  );
}

export default function SearchNavigation() {
  let [modifierKey, setModifierKey] = useState('');
  const [isSearchModalVisible, setIsSearchModalVisible] = useState(false);

  useEffect(() => {
    setModifierKey(/(Mac|iPhone|iPod|iPad)/i.test(navigator.platform) ? '⌘' : 'Ctrl');
  }, []);

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
      <button
        type="button"
        className="mr-4 flex items-center rounded-lg bg-slate-800 py-1 text-sm text-slate-400 ring-inset ring-white/10 transition hover:text-white hover:ring-white/20"
        onClick={() => setIsSearchModalVisible(true)}
        aria-label="Search by ID"
      >
        <div className="flex items-center gap-1 px-2">
          <MagnifyingGlassIcon className="h-4 w-4" />
          ID
        </div>

        <kbd className="flex items-center border-l border-slate-400 px-2">
          <kbd className="font-sans">{modifierKey}</kbd>
          <kbd className="font-sans">K</kbd>
        </kbd>
      </button>

      <SearchModal isOpen={isSearchModalVisible} onOpenChange={setIsSearchModalVisible} />
    </>
  );
}
