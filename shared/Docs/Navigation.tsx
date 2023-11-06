import { useRef } from "react";
import Link from "next/link";
import { useRouter } from "next/router";
import clsx from "clsx";
import { AnimatePresence, motion, useIsPresent } from "framer-motion";

import { Button } from "./Button";
import { useIsInsideMobileNavigation } from "./MobileNavigation";
import { useSectionStore } from "./SectionProvider";
import { Tag } from "./Tag";
import { remToPx } from "../../utils/remToPx";
import { topLevelNav } from "./navigationStructure";

const BASE_DIR = "/docs";

function useInitialValue(value, condition = true) {
  let initialValue = useRef(value).current;
  return condition ? initialValue : value;
}

function TopLevelNavItem({ href, children }) {
  return (
    <li className="lg:hidden">
      <Link
        href={href}
        className="block py-1 text-sm text-slate-600 transition hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
      >
        {children}
      </Link>
    </li>
  );
}

function NavLink({
  href,
  tag,
  active,
  isAnchorLink = false,
  isTopLevel = false,
  className = "",
  children,
  target,
}: {
  href: string;
  tag?: any;
  active?: boolean;
  isAnchorLink?: boolean;
  isTopLevel?: boolean;
  className?: string;
  target?: string;
  children: React.ReactNode;
}) {
  return (
    <LinkOrHref
      href={href}
      aria-current={active ? "page" : undefined}
      target={target}
      className={clsx(
        "flex justify-between items-center gap-2 py-1 pr-3 text-sm font-medium transition group", // group for nested hovers
        isTopLevel ? "pl-0" : isAnchorLink ? "pl-7" : "pl-4",
        active
          ? "text-slate-900 dark:text-white"
          : "text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white",
        className
      )}
    >
      <span className="truncate">{children}</span>
      {tag && (
        <Tag variant="small" color="slate">
          {tag}
        </Tag>
      )}
    </LinkOrHref>
  );
}

// LinkOrHref returns a standard link with target="_blank" if we want to open a docs
// link in a new tab.
const LinkOrHref = (props: any) => {
  if (props.target === "_blank") {
    return <a {...props} />;
  }
  return <Link {...props} />;
};

function VisibleSectionHighlight({ group, pathname }) {
  let [sections, visibleSections] = useInitialValue(
    [
      useSectionStore((s) => s.sections),
      useSectionStore((s) => s.visibleSections),
    ],
    useIsInsideMobileNavigation()
  );

  let isPresent = useIsPresent();
  let firstVisibleSectionIndex = Math.max(
    0,
    [{ id: "_top" }, ...sections].findIndex(
      (section) => section.id === visibleSections[0]
    )
  );
  let itemHeight = remToPx(1.76);
  let height = isPresent
    ? Math.max(1, visibleSections.length) * itemHeight
    : itemHeight;
  let top =
    group.links.findIndex((link) => link.href === pathname) * itemHeight +
    firstVisibleSectionIndex * itemHeight;

  return (
    <motion.div
      layout
      initial={{ opacity: 0 }}
      animate={{ opacity: 1, transition: { delay: 0.2 } }}
      exit={{ opacity: 0 }}
      className="absolute inset-x-0 top-0 bg-slate-800/2.5 will-change-transform dark:bg-white/2.5"
      style={{ borderRadius: 8, height, top }}
    />
  );
}

function ActivePageMarker({ group, pathname }) {
  let itemHeight = 28;
  let offset = remToPx(0.27);
  let activePageIndex = group.links.findIndex((link) => link.href === pathname);
  let top = offset + activePageIndex * itemHeight;

  return (
    <motion.div
      layout
      className="absolute left-2 h-[20px] w-px bg-indigo-500"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1, transition: { delay: 0.2 } }}
      exit={{ opacity: 0 }}
      style={{ top }}
    />
  );
}

// A nested navigation group of links that expand and follow
function NavigationGroup({ group, className = "" }) {
  // If this is the mobile navigation then we always render the initial
  // state, so that the state does not change during the close animation.
  // The state will still update when we re-open (re-render) the navigation.
  let isInsideMobileNavigation = useIsInsideMobileNavigation();
  let [router, sections] = useInitialValue(
    [useRouter(), useSectionStore((s) => s.sections)],
    isInsideMobileNavigation
  );

  let isActiveGroup =
    group.links.findIndex((link) => link.href === router.pathname) !== -1;

  return (
    <li className={clsx("relative mt-6", className)}>
      <motion.h2
        layout="position"
        className="text-xs font-semibold text-slate-900 dark:text-white uppercase font-mono"
      >
        {group.title}
      </motion.h2>
      <div className="relative mt-3 pl-2">
        <AnimatePresence initial={!isInsideMobileNavigation}>
          {isActiveGroup && (
            <VisibleSectionHighlight group={group} pathname={router.pathname} />
          )}
        </AnimatePresence>
        <motion.div
          layout
          className="absolute inset-y-0 left-2 w-px bg-slate-900/10 dark:bg-white/5"
        />
        <AnimatePresence initial={false}>
          {isActiveGroup && (
            <ActivePageMarker group={group} pathname={router.pathname} />
          )}
        </AnimatePresence>
        <ul role="list" className="border-l border-transparent">
          {group.links.map((link) => (
            <motion.li key={link.href} layout="position" className="relative">
              <NavLink
                href={link.href}
                active={link.href === router.pathname}
                className={link.className}
              >
                {link.title}
              </NavLink>
              <AnimatePresence mode="popLayout" initial={false}>
                {link.href === router.pathname && sections.length > 0 && (
                  <motion.ul
                    role="list"
                    initial={{ opacity: 0 }}
                    animate={{
                      opacity: 1,
                      transition: { delay: 0.1 },
                    }}
                    exit={{
                      opacity: 0,
                      transition: { duration: 0.15 },
                    }}
                  >
                    {sections.map((section) => (
                      <li key={section.id}>
                        <NavLink
                          href={`${link.href}#${section.id}`}
                          tag={section.tag}
                          isAnchorLink
                        >
                          {section.title}
                        </NavLink>
                      </li>
                    ))}
                  </motion.ul>
                )}
              </AnimatePresence>
            </motion.li>
          ))}
        </ul>
      </div>
    </li>
  );
}

export const headerLinks = [
  {
    title: "Docs",
    href: BASE_DIR,
  },
  {
    title: "Patterns",
    href: "/patterns?ref=docs",
  },
];

// Flatten the nested nav and get all nav sections w/ sectionLinks
function getAllSections(nav) {
  return nav.reduce((acc, item) => {
    if (item.sectionLinks) {
      acc.push(item);
    }
    if (item.links) {
      acc.push(...getAllSections(item.links));
    }
    return acc;
  }, []);
}

function findRecursiveSectionLinkMatch(nav, pathname) {
  const sections = getAllSections(nav);
  return sections.find(({ matcher, sectionLinks }) => {
    if (matcher?.test(pathname)) {
      return true;
    }
    return !!sectionLinks?.find((item) => {
      return item.links?.find((link) => link.href === pathname);
    });
  });
}
// todo fix active on top level

export function Navigation(props) {
  const router = useRouter();
  // Remove query params and hash from pathname
  const pathname = router.asPath.replace(/(\?|#).+$/, "");

  const nestedSection = findRecursiveSectionLinkMatch(topLevelNav, pathname);
  const isNested = !!nestedSection;
  const nestedNavigation = nestedSection;

  return (
    <nav {...props}>
      {isNested && (
        <NavLink href={BASE_DIR} className="pl-0 text-xs uppercase font-mono">
          ← Back to docs home
        </NavLink>
      )}
      <ul role="list" className={!isNested ? "flex flex-col gap-2" : undefined}>
        {nestedNavigation ? (
          <>
            <li className="mt-6 mb-4 flex gap-2 items-center text-base font-semibold text-slate-900 dark:text-white">
              <span className="p-0.5">
                <nestedNavigation.icon className="w-5 h-5 text-slate-400" />
              </span>
              {nestedNavigation.title}
            </li>
            {nestedNavigation.sectionLinks.map((group, groupIndex) => (
              <NavigationGroup key={group.title} group={group} />
            ))}
          </>
        ) : (
          topLevelNav.map((item, idx) =>
            item.href ? (
              <li key={idx}>
                <NavLink href={item.href} key={idx} isTopLevel={true}>
                  <span className="flex flex-row gap-3 items-center">
                    {item.icon && (
                      <item.icon className="w-5 h-5 text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-200" />
                    )}
                    {item.title}
                  </span>
                </NavLink>
              </li>
            ) : (
              <li className="mt-6" key={idx}>
                <h2 className="text-xs font-semibold text-slate-900 dark:text-white uppercase">
                  {item.title}
                </h2>
                <ul role="list" className="mt-3 flex flex-col gap-2">
                  {item.links.map((link, idx) => (
                    <li key={idx}>
                      <NavLink
                        href={link.href}
                        isTopLevel={true}
                        tag={link.tag}
                        target={link.target}
                      >
                        <span className="flex flex-row gap-3 items-center">
                          {link.icon && (
                            <link.icon className="w-5 h-4 text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-200" />
                          )}
                          {link.title}
                        </span>
                      </NavLink>
                    </li>
                  ))}
                </ul>
              </li>
            )
          )
        )}

        <li className="sticky bottom-0 z-10 mt-6 sm:hidden gap-2 flex dark:bg-slate-900">
          <Button
            href={process.env.NEXT_PUBLIC_SIGNIN_URL}
            variant="secondary"
            className="w-full"
          >
            Sign In
          </Button>
          <Button
            href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=docs-mobile-nav`}
            variant="primary"
            arrow="right"
            className="w-full"
          >
            Sign Up
          </Button>
        </li>
      </ul>
    </nav>
  );
}
