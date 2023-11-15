'use client';

import { useEffect, useRef, useState } from 'react';
import { type Route } from 'next';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useUser } from '@clerk/nextjs';
import { ArrowUturnRightIcon } from '@heroicons/react/20/solid';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { getEnvironmentSlug } from '@/utils/environments';
import Modal from '../Modal';

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
  onClose: () => void;
};

function SearchModal({ isOpen, onClose }: SearchModalProps) {
  const [isSearchResultsListOpened, setIsSearchResultsListOpened] = useState(false);
  const [search, setSearch] = useState('');
  const resultRef = useRef(null);
  const router = useRouter();
  const { user } = useUser();
  let searchResult = {
    type: '',
    href: '',
    name: '',
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
        globalResults?.value.__typename === 'FunctionRun' ? globalResults?.value?.functionID : '',
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
    };
  } else if (globalResults?.value.__typename === 'ArchivedEvent') {
    searchResult = {
      type: 'event',
      href: `/env/${environmentSlug}/events/${encodeURIComponent(globalResults.value?.name)}/logs/${
        globalResults.value.id
      }`,
      name: globalResults.value?.name,
    };
  }

  /*
   * Event handlers
   */
  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.code === 'Enter' && resultRef.current && searchResult.href.length > 0) {
      router.push(searchResult.href as Route);
      onClose();
    }
  }

  let debounce: NodeJS.Timeout;
  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    clearTimeout(debounce);
    if (e.target.value === '') {
      setIsSearchResultsListOpened(false);
      return setSearch('');
    }
    debounce = setTimeout(() => {
      setIsSearchResultsListOpened(true);
      setSearch(e.target.value);
    }, 500);
  }

  return (
    <Modal
      backdropClassName="bg-white/30 backdrop-blur-[2px]"
      className="ml-auto mr-auto flex max-w-2xl self-start p-0 shadow"
      isOpen={isOpen}
      onClose={onClose}
    >
      <form className="w-full divide-y divide-slate-100 ">
        <div className="flex items-center gap-2 px-4 py-3">
          <input
            className="w-[34rem] placeholder-slate-500 focus-visible:outline-none"
            placeholder="Search by ID..."
            autoFocus
            onChange={handleChange}
            defaultValue={search}
            onKeyDown={handleKeyDown}
          />
        </div>
        {isSearchResultsListOpened && (
          <>
            {!globalResults ? (
              <div className="px-4 py-3 text-sm text-slate-600">
                {!isFetching && 'Nothing found. Make sure you are typing the full ID.'}
                {isFetching && 'Searching...'}
              </div>
            ) : (
              <ul role="listbox">
                <li
                  role="option"
                  aria-selected="true"
                  className="group aria-selected:bg-slate-100"
                  ref={resultRef}
                >
                  <Link
                    onClick={onClose}
                    href={searchResult.href as Route}
                    className="flex items-center px-4 py-3"
                  >
                    <div>
                      <div>{searchResult.name}</div>
                      <div className="mt-1 text-xs font-medium capitalize text-slate-400">
                        {searchResult.type}
                      </div>
                    </div>

                    <kbd
                      aria-label="press enter to jump to page"
                      className="ml-auto hidden rounded bg-slate-500 p-2 text-white group-aria-selected:block"
                    >
                      <ArrowUturnRightIcon className="h-3 w-3 rotate-180" />
                    </kbd>
                  </Link>
                </li>
              </ul>
            )}
          </>
        )}
      </form>
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
    if (isSearchModalVisible) {
      return;
    }

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setIsSearchModalVisible(true);
      }
      if (e.key === 'escape') {
        e.preventDefault();
        setIsSearchModalVisible(false);
      }
    }

    window.addEventListener('keydown', onKeyDown);

    return () => {
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [isSearchModalVisible, setIsSearchModalVisible]);

  return (
    <>
      <div className="ml-auto mr-4 block max-w-xs flex-auto">
        <button
          type="button"
          className="flex h-8 w-full items-center gap-2 rounded-lg bg-slate-800 pl-2 pr-3 text-sm text-slate-400 ring-inset ring-white/10 transition hover:text-white hover:ring-white/20"
          onClick={() => setIsSearchModalVisible(true)}
        >
          Search by ID...
          <kbd className="ml-auto flex items-center gap-1">
            <kbd className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-600 font-sans text-xs text-white">
              {modifierKey}
            </kbd>
            <kbd className="ml-auto flex h-6 w-6 items-center justify-center rounded bg-slate-600 font-sans text-xs text-white">
              K
            </kbd>
          </kbd>
        </button>
      </div>
      <SearchModal isOpen={isSearchModalVisible} onClose={() => setIsSearchModalVisible(false)} />
    </>
  );
}
