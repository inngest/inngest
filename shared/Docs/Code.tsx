import React, {
  Children,
  createContext,
  ReactNode,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { useRouter } from "next/router";
import { Tab } from "@headlessui/react";
import clsx from "clsx";
import create from "zustand";

import { Tag } from "./Tag";

const languageNames = {
  js: "JavaScript",
  ts: "TypeScript",
  javascript: "JavaScript",
  typescript: "TypeScript",
  php: "PHP",
  python: "Python",
  ruby: "Ruby",
  go: "Go",
};

function getPanelTitle({ title, language }) {
  return title ?? languageNames[language] ?? "Code";
}

function ClipboardIcon(props) {
  return (
    <svg viewBox="0 0 20 20" aria-hidden="true" {...props}>
      <path
        strokeWidth="0"
        d="M5.5 13.5v-5a2 2 0 0 1 2-2l.447-.894A2 2 0 0 1 9.737 4.5h.527a2 2 0 0 1 1.789 1.106l.447.894a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-5a2 2 0 0 1-2-2Z"
      />
      <path
        fill="none"
        strokeLinejoin="round"
        d="M12.5 6.5a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-5a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2m5 0-.447-.894a2 2 0 0 0-1.79-1.106h-.527a2 2 0 0 0-1.789 1.106L7.5 6.5m5 0-1 1h-3l-1-1"
      />
    </svg>
  );
}

function CopyButton({ code }) {
  let [copyCount, setCopyCount] = useState(0);
  let copied = copyCount > 0;

  useEffect(() => {
    if (copyCount > 0) {
      let timeout = setTimeout(() => setCopyCount(0), 1000);
      return () => {
        clearTimeout(timeout);
      };
    }
  }, [copyCount]);

  return (
    <button
      type="button"
      className={clsx(
        "group/button absolute top-3 right-4 overflow-hidden rounded-full py-1 pl-2 pr-3 text-2xs font-medium opacity-0 backdrop-blur transition focus:opacity-100 group-hover:opacity-100",
        copied
          ? "bg-indigo-400/10 ring-1 ring-inset ring-indigo-400/20"
          : "bg-white/5 hover:bg-white/7.5 dark:bg-white/2.5 dark:hover:bg-white/5"
      )}
      onClick={() => {
        window.navigator.clipboard.writeText(code).then(() => {
          setCopyCount((count) => count + 1);
        });
      }}
    >
      <span
        aria-hidden={copied}
        className={clsx(
          "pointer-events-none flex items-center gap-0.5 text-slate-400 transition duration-300",
          copied && "-translate-y-1.5 opacity-0"
        )}
      >
        <ClipboardIcon className="h-5 w-5 fill-slate-500/20 stroke-slate-500 transition-colors group-hover/button:stroke-slate-400" />
        Copy
      </span>
      <span
        aria-hidden={!copied}
        className={clsx(
          "pointer-events-none absolute inset-0 flex items-center justify-center text-indigo-400 transition duration-300",
          !copied && "translate-y-1.5 opacity-0"
        )}
      >
        Copied!
      </span>
    </button>
  );
}

function CodePanelHeader({ tag, label }) {
  if (!tag && !label) {
    return null;
  }

  return (
    <div className="flex h-9 items-center gap-2 border-y border-t-transparent border-b-white/7.5 bg-slate-900 bg-white/2.5 px-4 dark:border-b-white/5 dark:bg-white/1">
      {tag && (
        <div className="dark flex">
          <Tag variant="small">{tag}</Tag>
        </div>
      )}
      {tag && label && (
        <span className="h-0.5 w-0.5 rounded-full bg-slate-500" />
      )}
      {label && (
        <span className="font-mono text-xs text-slate-400">{label}</span>
      )}
    </div>
  );
}

type CodePanelProps = {
  tag?: string;
  label?: string;
  code?: string;
  children?: React.ReactNode;
};

function CodePanel({ tag, label, code, children }: CodePanelProps) {
  let child = Children.only<any>(children);

  return (
    <div className="group dark:bg-white/2.5">
      <CodePanelHeader
        tag={child.props.tag ?? tag}
        label={child.props.label ?? label}
      />
      <div className="relative">
        <pre className="overflow-x-auto px-6 py-5 text-xs text-white leading-relaxed">
          {children}
        </pre>
        <CopyButton code={child.props.code ?? code} />
      </div>
    </div>
  );
}

type CodeGroupHeaderProps = {
  title?: string;
  filename?: string;
  hasTabs?: boolean;
  children: React.ReactNode;
  selectedIndex?: number;
};

function CodeGroupHeader({
  title,
  filename,
  children,
  hasTabs,
  selectedIndex,
}: CodeGroupHeaderProps) {
  const heading = title || filename;

  if (!heading && !hasTabs) {
    return null;
  }

  return (
    <div className="px-6 gap-x-4 bg-slate-800 flex min-h-[calc(theme(spacing.10)+1px)] flex-wrap items-center dark:bg-transparent">
      {heading && (
        <h3
          className={clsx(
            "mr-auto text-xs font-semibold text-white",
            !!filename && "font-mono"
          )}
        >
          {filename ? <code>{heading}</code> : heading}
        </h3>
      )}
      {hasTabs && (
        <Tab.List className="-mb-px flex gap-4 text-xs font-medium">
          {Children.map<ReactNode, any>(children, (child, childIndex) => (
            <Tab
              className={clsx(
                "border-b py-3 transition focus:outline-none",
                childIndex === selectedIndex
                  ? "border-indigo-500 text-indigo-400"
                  : "border-transparent text-slate-400 hover:text-slate-300"
              )}
            >
              {getPanelTitle(child.props)}
            </Tab>
          ))}
        </Tab.List>
      )}
    </div>
  );
}

function CodeGroupPanels({ hasTabs, children, ...props }) {
  if (hasTabs) {
    return (
      <Tab.Panels>
        {Children.map(children, (child) => (
          <Tab.Panel>
            <CodePanel {...props}>{child}</CodePanel>
          </Tab.Panel>
        ))}
      </Tab.Panels>
    );
  }

  return <CodePanel {...props}>{children}</CodePanel>;
}

function usePreventLayoutShift() {
  let positionRef = useRef<HTMLElement>();
  let rafRef = useRef<number>();

  useEffect(() => {
    return () => {
      window.cancelAnimationFrame(rafRef.current);
    };
  }, []);

  return {
    positionRef,
    preventLayoutShift(callback) {
      let initialTop = positionRef.current?.getBoundingClientRect().top;

      callback();

      rafRef.current = window.requestAnimationFrame(() => {
        let newTop = positionRef.current.getBoundingClientRect().top;
        window.scrollBy(0, newTop - initialTop);
      });
    },
  };
}

type PreferredLanguageStore = {
  preferredLanguages: string[];
  addPreferredLanguage: (string) => void;
};
const usePreferredLanguageStore = create<PreferredLanguageStore>((set) => ({
  preferredLanguages: [],
  addPreferredLanguage: (language) =>
    set((state) => ({
      preferredLanguages: [
        ...state.preferredLanguages.filter(
          (preferredLanguage) => preferredLanguage !== language
        ),
        language,
      ],
    })),
}));

function useTabGroupProps(availableLanguages) {
  let { preferredLanguages, addPreferredLanguage } =
    usePreferredLanguageStore();
  let [selectedIndex, setSelectedIndex] = useState(0);
  let activeLanguage = [...availableLanguages].sort(
    (a, z) => preferredLanguages.indexOf(z) - preferredLanguages.indexOf(a)
  )[0];
  let languageIndex = availableLanguages.indexOf(activeLanguage);
  let newSelectedIndex = languageIndex === -1 ? selectedIndex : languageIndex;
  if (newSelectedIndex !== selectedIndex) {
    setSelectedIndex(newSelectedIndex);
  }

  let { positionRef, preventLayoutShift } = usePreventLayoutShift();

  return {
    as: "div",
    ref: positionRef,
    selectedIndex,
    onChange: (newSelectedIndex) => {
      preventLayoutShift(() =>
        addPreferredLanguage(availableLanguages[newSelectedIndex])
      );
    },
  };
}

const CodeGroupContext = createContext(false);

type CodeGroupProps = {
  title?: string;
  filename?: string;
  forceTabs?: boolean;
  children: React.ReactNode;
};

export function CodeGroup({
  children,
  title,
  filename,
  forceTabs,
  ...props
}: CodeGroupProps) {
  let languages = Children.map<string, any>(children, (child) =>
    getPanelTitle(child.props)
  );
  let tabGroupProps = useTabGroupProps(languages);
  let hasTabs = forceTabs || Children.count(children) > 1;
  let Container: typeof Tab["Group"] | "div" = hasTabs ? Tab.Group : "div";
  let containerProps = hasTabs ? tabGroupProps : {};
  let headerProps = hasTabs
    ? { selectedIndex: tabGroupProps.selectedIndex }
    : {};

  return (
    <CodeGroupContext.Provider value={true}>
      <Container
        {...containerProps}
        className="not-prose my-6 overflow-hidden rounded-lg bg-slate-900 shadow-md"
      >
        <CodeGroupHeader
          title={title}
          filename={filename}
          hasTabs={hasTabs}
          {...headerProps}
        >
          {children}
        </CodeGroupHeader>
        <CodeGroupPanels hasTabs={hasTabs} {...props}>
          {children}
        </CodeGroupPanels>
      </Container>
    </CodeGroupContext.Provider>
  );
}

export function Code({ children, ...props }) {
  let isGrouped = useContext(CodeGroupContext);

  if (isGrouped) {
    return <code {...props} dangerouslySetInnerHTML={{ __html: children }} />;
  }

  return <code {...props}>{children}</code>;
}

export function Pre({ children, ...props }) {
  let isGrouped = useContext(CodeGroupContext);

  if (isGrouped) {
    return children;
  }

  return <CodeGroup {...props}>{children}</CodeGroup>;
}

type GuideOption = {
  key: string;
  title: string;
};

const GuideSelectorContext = createContext<{
  selected: string;
  options: GuideOption[];
}>(null);

export function GuideSelector({
  children,
  options = [],
}: {
  children: React.ReactNode;
  options: GuideOption[];
}) {
  const router = useRouter();
  const searchParamKey = "guide";
  const [selected, setSelected] = useState<string>(options[0].key);

  useEffect(() => {
    const urlSelected = Array.isArray(router.query[searchParamKey])
      ? router.query[searchParamKey][0]
      : router.query[searchParamKey];
    const isValidOption = options.find((o) => o.key === urlSelected);
    if (isValidOption && Boolean(urlSelected) && urlSelected !== selected) {
      setSelected(urlSelected);
    }
  }, [router, selected]);

  const onChange = (newSelectedIndex) => {
    const newSelectedKey = options[newSelectedIndex].key;
    setSelected(newSelectedKey);
    const url = new URL(router.asPath, window.location.origin);
    url.searchParams.set(searchParamKey, newSelectedKey);
    // Replace the URL state and do use shallow to avoid refresh
    router.replace(url.toString(), null, { shallow: true, scroll: false });
  };

  return (
    <GuideSelectorContext.Provider value={{ selected, options }}>
      <Tab.Group onChange={onChange}>
        <Tab.List className="-mb-px flex gap-4 text-sm font-medium">
          {options.map((option, idx) => (
            <Tab
              key={idx}
              className={clsx(
                "border-b py-3 transition focus:outline-none",
                option.key === selected
                  ? "border-indigo-500 text-indigo-700"
                  : "border-transparent text-slate-800 hover:text-indigo-600"
              )}
            >
              {option.title}
            </Tab>
          ))}
        </Tab.List>
      </Tab.Group>
      {children}
    </GuideSelectorContext.Provider>
  );
}

export function GuideSection({
  children,
  show,
}: {
  children: React.ReactNode;
  show: string;
}) {
  let context = useContext(GuideSelectorContext);
  if (show === context.selected) {
    return <>{children}</>;
  }
  return null;
}
