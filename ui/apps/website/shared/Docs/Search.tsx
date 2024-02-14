import {
  forwardRef,
  Fragment,
  MutableRefObject,
  ReactElement,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";
import { useRouter } from "next/router";
import {
  createAutocomplete,
  AutocompleteApi,
} from "@algolia/autocomplete-core";
import { getAlgoliaResults } from "@algolia/autocomplete-preset-algolia";
import { Dialog, Transition } from "@headlessui/react";
import algoliasearch from "algoliasearch/lite";
import clsx from "clsx";

import PythonIcon from "src/shared/Icons/Python";
import TypeScriptIcon from "src/shared/Icons/TypeScript";

const searchClient = algoliasearch(
  process.env.NEXT_PUBLIC_DOCSEARCH_APP_ID,
  process.env.NEXT_PUBLIC_DOCSEARCH_API_KEY
);

type AutocompleteItem = {
  url: string;
  query: string;
  [key: string]: any;
};
type AutocompleteState = {
  status?: string;
  isOpen?: boolean;
  query?: string;
  collections?: any[];
};

function useAutocomplete() {
  let id = useId();
  let router = useRouter();
  let [autocompleteState, setAutocompleteState] = useState<AutocompleteState>(
    {}
  );

  let [autocomplete] = useState(() =>
    createAutocomplete<AutocompleteItem>({
      id,
      placeholder: "Find something...",
      defaultActiveItemId: 0,
      onStateChange({ state }) {
        setAutocompleteState(state);
      },
      shouldPanelOpen({ state }) {
        return state.query !== "";
      },
      navigator: {
        navigate({ itemUrl }) {
          autocomplete.setIsOpen(true);
          router.push(itemUrl);
        },
      },
      getSources() {
        return [
          {
            sourceId: "documentation",
            getItemInputValue({ item }) {
              return item.query;
            },
            getItemUrl({ item }) {
              let url = new URL(item.url);
              return `${url.pathname}${url.hash}`;
            },
            onSelect({ itemUrl }) {
              router.push(itemUrl);
            },
            getItems({ query }) {
              return getAlgoliaResults({
                searchClient,
                queries: [
                  {
                    query,
                    indexName: process.env.NEXT_PUBLIC_DOCSEARCH_INDEX_NAME,
                    params: {
                      hitsPerPage: 5,
                      highlightPreTag:
                        '<mark class="underline bg-transparent text-indigo-500">',
                      highlightPostTag: "</mark>",
                    },
                  },
                ],
              });
            },
          },
        ];
      },
    })
  );

  return { autocomplete, autocompleteState };
}

type Result = {
  type: string;
  hierarchy: {
    [key: string]: any;
  };
  _highlightResult: {
    hierarchy: {
      [key: string]: any;
    };
  };
};

function resolveResult(result: Result): {
  titleHtml: string;
  hierarchyHtml: string[];
} {
  let allLevels = Object.keys(result.hierarchy);
  let hierarchy = Object.entries(result._highlightResult.hierarchy).filter(
    ([, { value }]) => Boolean(value)
  );
  let levels = hierarchy.map(([level]) => level);

  let level =
    result.type === "content"
      ? levels.pop()
      : levels
          .filter(
            (level) =>
              allLevels.indexOf(level) <= allLevels.indexOf(result.type)
          )
          .pop();

  return {
    titleHtml: result._highlightResult.hierarchy[level].value,
    hierarchyHtml: hierarchy
      .slice(0, levels.indexOf(level))
      .map(([, { value }]) => value),
  };
}

function SearchIcon(props) {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" {...props}>
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12.01 12a4.25 4.25 0 1 0-6.02-6 4.25 4.25 0 0 0 6.02 6Zm0 0 3.24 3.25"
      />
    </svg>
  );
}

function NoResultsIcon(props) {
  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" {...props}>
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12.01 12a4.237 4.237 0 0 0 1.24-3c0-.62-.132-1.207-.37-1.738M12.01 12A4.237 4.237 0 0 1 9 13.25c-.635 0-1.237-.14-1.777-.388M12.01 12l3.24 3.25m-3.715-9.661a4.25 4.25 0 0 0-5.975 5.908M4.5 15.5l11-11"
      />
    </svg>
  );
}

function LoadingIcon(props) {
  let id = useId();

  return (
    <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" {...props}>
      <circle cx="10" cy="10" r="5.5" strokeLinejoin="round" />
      <path
        stroke={`url(#${id})`}
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.5 10a5.5 5.5 0 1 0-5.5 5.5"
      />
      <defs>
        <linearGradient
          id={id}
          x1="13"
          x2="9.5"
          y1="9"
          y2="15"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="currentColor" />
          <stop offset="1" stopColor="currentColor" stopOpacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
}

function getSDKLanguage(result): string | null {
  switch (result.sdkLanguage) {
    case "typescript":
      return "TypeScript";
    case "python":
      return "Python";
    default:
      return null;
  }
}
function getSDKLanguageIcon(result): React.ElementType | null {
  switch (result.sdkLanguage) {
    case "typescript":
      return TypeScriptIcon;
    case "python":
      return PythonIcon;
    default:
      return null;
  }
}

function SearchResult({ result, resultIndex, autocomplete, collection }) {
  let id = useId();
  let { titleHtml, hierarchyHtml } = resolveResult(result);

  const sdkLanguage = getSDKLanguage(result);
  const SdkLanguageIcon = getSDKLanguageIcon(result);

  return (
    <li
      className={clsx(
        "group block relative cursor-default px-4 py-3 aria-selected:bg-slate-50 dark:aria-selected:bg-slate-800/50",
        resultIndex > 0 && "border-t border-slate-100 dark:border-slate-800"
      )}
      aria-labelledby={`${id}-hierarchy ${id}-title`}
      {...autocomplete.getItemProps({
        item: result,
        source: collection.source,
      })}
    >
      <div
        id={`${id}-title`}
        aria-hidden="true"
        className="text-sm font-medium text-slate-900 group-aria-selected:text-indigo-500 dark:text-white"
        dangerouslySetInnerHTML={{ __html: titleHtml }}
      />
      {SdkLanguageIcon && (
        <span className="absolute px-1.5 top-3 right-2">
          <SdkLanguageIcon className="w-5 h-5 text-slate-400" />
        </span>
      )}

      {hierarchyHtml.length > 0 && (
        <div
          id={`${id}-hierarchy`}
          aria-hidden="true"
          className="mt-1 truncate whitespace-nowrap text-2xs text-slate-500"
        >
          {hierarchyHtml.map((item, itemIndex, items) => (
            <Fragment key={itemIndex}>
              <span dangerouslySetInnerHTML={{ __html: item }} />
              <span
                className={
                  itemIndex === items.length - 1
                    ? "sr-only"
                    : "mx-2 text-slate-300 dark:text-slate-700"
                }
              >
                /
              </span>
            </Fragment>
          ))}
        </div>
      )}
    </li>
  );
}

function SearchResults({ autocomplete, query, collection }) {
  if (collection.items.length === 0) {
    return (
      <div className="p-6 text-center">
        <NoResultsIcon className="mx-auto h-5 w-5 stroke-slate-900 dark:stroke-slate-600" />
        <p className="mt-2 text-xs text-slate-700 dark:text-slate-400">
          Nothing found for{" "}
          <strong className="break-words font-semibold text-slate-900 dark:text-white">
            &lsquo;{query}&rsquo;
          </strong>
          . Please try again.
        </p>
      </div>
    );
  }

  return (
    <ul role="list" {...autocomplete.getListProps()}>
      {collection.items.map((result, resultIndex) => (
        <SearchResult
          key={result.objectID}
          result={result}
          resultIndex={resultIndex}
          autocomplete={autocomplete}
          collection={collection}
        />
      ))}
    </ul>
  );
}

type SearchInputProps = {
  autocomplete: AutocompleteApi<
    AutocompleteItem,
    Event,
    MouseEvent,
    KeyboardEvent
  >;
  autocompleteState: AutocompleteState;
  onClose: () => void;
};

const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput(
    { autocomplete, autocompleteState, onClose }: SearchInputProps,
    inputRef
  ) {
    // Modified to fix ts
    let inputProps = autocomplete.getInputProps({} as any);

    return (
      <div className="group relative flex h-12">
        <SearchIcon className="pointer-events-none absolute left-3 top-0 h-full w-5 stroke-slate-500" />
        {/* @ts-ignore */}
        <input
          ref={inputRef}
          className={clsx(
            "flex-auto appearance-none bg-transparent pl-10 text-slate-900 outline-none placeholder:text-slate-500 focus:w-full focus:flex-none dark:text-white sm:text-sm [&::-webkit-search-cancel-button]:hidden [&::-webkit-search-decoration]:hidden [&::-webkit-search-results-button]:hidden [&::-webkit-search-results-decoration]:hidden",
            autocompleteState.status === "stalled" ? "pr-11" : "pr-4"
          )}
          {...inputProps}
          onKeyDown={(event) => {
            if (
              event.key === "Escape" &&
              !autocompleteState.isOpen &&
              autocompleteState.query === ""
            ) {
              onClose();
            } else {
              // @ts-ignore
              inputProps.onKeyDown(event);
            }
          }}
        />
        {autocompleteState.status === "stalled" && (
          <div className="absolute inset-y-0 right-3 flex items-center">
            <LoadingIcon className="h-5 w-5 animate-spin stroke-slate-200 text-slate-900 dark:stroke-slate-800 dark:text-indigo-400" />
          </div>
        )}
      </div>
    );
  }
);

function SearchDialog({ open, setOpen, className }: SearchDialogProps) {
  let router = useRouter();
  let formRef = useRef();
  let panelRef = useRef();
  let inputRef = useRef<HTMLInputElement>();
  let { autocomplete, autocompleteState } = useAutocomplete();

  useEffect(() => {
    if (!open) {
      return;
    }

    function onRouteChange() {
      setOpen(false);
    }

    router.events.on("routeChangeStart", onRouteChange);
    router.events.on("hashChangeStart", onRouteChange);

    return () => {
      router.events.off("routeChangeStart", onRouteChange);
      router.events.off("hashChangeStart", onRouteChange);
    };
  }, [open, setOpen, router]);

  useEffect(() => {
    if (open) {
      return;
    }

    function onKeyDown(event) {
      if (event.key === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        setOpen(true);
      }
    }

    window.addEventListener("keydown", onKeyDown);

    return () => {
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [open, setOpen]);

  function onClose(open) {
    setOpen(open);
  }

  return (
    <Transition.Root
      show={open}
      as={Fragment}
      afterLeave={() => autocomplete.setQuery("")}
    >
      <Dialog
        onClose={onClose}
        className={clsx("fixed inset-0 z-50", className)}
      >
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-slate-400/25 backdrop-blur-sm dark:bg-black/40" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto px-4 py-4 sm:py-20 sm:px-6 md:py-32 lg:px-8 lg:py-[15vh]">
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-300"
            enterFrom="opacity-0 scale-95"
            enterTo="opacity-100 scale-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100 scale-100"
            leaveTo="opacity-0 scale-95"
          >
            <Dialog.Panel className="mx-auto overflow-hidden rounded-lg bg-slate-50 shadow-xl ring-1 ring-slate-900/7.5 dark:bg-slate-900 dark:ring-slate-800 sm:max-w-xl">
              <div {...autocomplete.getRootProps({})}>
                {/* @ts-ignore */}
                <form
                  ref={formRef}
                  {...autocomplete.getFormProps({
                    inputElement: inputRef.current,
                  })}
                >
                  <SearchInput
                    ref={inputRef}
                    autocomplete={autocomplete}
                    autocompleteState={autocompleteState}
                    onClose={() => setOpen(false)}
                  />
                  {/* @ts-ignore */}
                  <div
                    ref={panelRef}
                    className="border-t border-slate-200 bg-white empty:hidden dark:border-slate-100/5 dark:bg-white/2.5"
                    {...autocomplete.getPanelProps({})}
                  >
                    {autocompleteState.isOpen && (
                      <>
                        <SearchResults
                          autocomplete={autocomplete}
                          query={autocompleteState.query}
                          collection={autocompleteState.collections[0]}
                        />
                      </>
                    )}
                  </div>
                </form>
              </div>
            </Dialog.Panel>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition.Root>
  );
}

type SearchButtonProps = {
  ref: MutableRefObject<HTMLButtonElement>;
  onClick: () => void;
};
type SearchDialogProps = {
  open: boolean;
  setOpen: (boolean) => void;
  className?: string;
};
type SearchProps = {
  buttonProps: SearchButtonProps;
  dialogProps: SearchDialogProps;
};

function useSearchProps(): SearchProps {
  let buttonRef = useRef<HTMLButtonElement>();
  let [open, setOpen] = useState(false);

  return {
    buttonProps: {
      ref: buttonRef,
      onClick() {
        setOpen(true);
      },
    },
    dialogProps: {
      open,
      setOpen(open) {
        let { width, height } = buttonRef.current.getBoundingClientRect();
        if (!open || (width !== 0 && height !== 0)) {
          setOpen(open);
        }
      },
    },
  };
}

export function Search() {
  let [modifierKey, setModifierKey] = useState<string>();
  let { buttonProps, dialogProps } = useSearchProps();

  useEffect(() => {
    setModifierKey(
      /(Mac|iPhone|iPod|iPad)/i.test(navigator.platform) ? "⌘" : "Ctrl "
    );
  }, []);

  return (
    <div className="hidden lg:block lg:max-w-md lg:flex-auto">
      <button
        type="button"
        className="hidden h-8 w-full items-center gap-2 rounded-full bg-white pl-2 pr-3 text-sm text-slate-500 ring-1 ring-slate-900/10 transition hover:ring-slate-900/20 dark:bg-white/5 dark:text-slate-400 dark:ring-inset dark:ring-white/10 dark:hover:ring-white/20 lg:flex focus:outline-none"
        {...buttonProps}
      >
        <SearchIcon className="h-5 w-5 stroke-current" />
        Search...
        <kbd className="ml-auto text-xs font-sans text-slate-500 dark:text-slate-400">
          {modifierKey}K
        </kbd>
      </button>
      <SearchDialog className="hidden lg:block" {...dialogProps} />
    </div>
  );
}

export function MobileSearch() {
  let { buttonProps, dialogProps } = useSearchProps();

  return (
    <div className="contents lg:hidden">
      <button
        type="button"
        className="flex h-6 w-6 items-center justify-center rounded-md transition hover:bg-slate-900/5 dark:hover:bg-white/5 lg:hidden focus:[&:not(:focus-visible)]:outline-none"
        aria-label="Find something..."
        {...buttonProps}
      >
        <SearchIcon className="h-5 w-5 stroke-slate-900 dark:stroke-white" />
      </button>
      <SearchDialog className="lg:hidden" {...dialogProps} />
    </div>
  );
}
