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
import { IconDocs, IconGuide } from "../Icons/duotone";

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
  children,
}: {
  href: string;
  tag?: any;
  active?: boolean;
  isAnchorLink?: boolean;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      aria-current={active ? "page" : undefined}
      className={clsx(
        "flex justify-between gap-2 py-1 pr-3 text-sm transition",
        isAnchorLink ? "pl-7" : "pl-4",
        active
          ? "text-slate-900 dark:text-white"
          : "text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
      )}
    >
      <span className="truncate">{children}</span>
      {tag && (
        <Tag variant="small" color="slate">
          {tag}
        </Tag>
      )}
    </Link>
  );
}

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

function NavigationGroup({ group, className }) {
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
        className="text-xs font-semibold text-slate-900 dark:text-white"
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
              <NavLink href={link.href} active={link.href === router.pathname}>
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

const baseDir = "/docs";
export const navigation = [
  {
    title: "Introduction",
    links: [
      { title: "Overview", href: `${baseDir}` },
      { title: "Quick Start Tutorial", href: `${baseDir}/quick-start` },
    ],
  },
  {
    title: "Getting Started",
    links: [
      { title: "SDK Overview", href: `${baseDir}/sdk/overview` },
      { title: "Serving the API & Frameworks", href: `${baseDir}/sdk/serve` },
      { title: "Writing Functions", href: `${baseDir}/functions` },
      { title: "Sending Events", href: `${baseDir}/events` },
      {
        title: "Multi-step Functions",
        href: `${baseDir}/functions/multi-step`,
      },
      {
        title: "Local Development",
        href: `${baseDir}/local-development`,
      },
    ],
  },
  {
    title: "Use Cases",
    links: [
      {
        title: "Background jobs",
        href: `${baseDir}/guides/background-jobs`,
      },
      {
        title: "Enqueueing future jobs",
        href: `${baseDir}/guides/enqueueing-future-jobs`,
      },
      {
        title: "Scheduled functions",
        href: `${baseDir}/guides/scheduled-functions`,
      },
      {
        title: "Step parallelism",
        href: `${baseDir}/guides/step-parallelism`,
      },
      {
        title: "Fan-out jobs",
        href: `${baseDir}/guides/fan-out-jobs`,
      },
      {
        title: "Trigger code from Retool",
        href: `${baseDir}/guides/trigger-your-code-from-retool`,
      },
      {
        title: "Instrumenting GraphQL",
        href: `${baseDir}/guides/instrumenting-graphql`,
      },
    ],
  },
  {
    title: "Platform",
    links: [
      {
        title: "Working With Environments",
        href: `${baseDir}/platform/environments`,
      },
      {
        title: "Creating an Event Key",
        href: `${baseDir}/events/creating-an-event-key`,
      },
      { title: "How to Deploy", href: `${baseDir}/deploy` },
      { title: "Deploy: Vercel", href: `${baseDir}/deploy/vercel` },
      { title: "Deploy: Netlify", href: `${baseDir}/deploy/netlify` },
      {
        title: "Deploy: Cloudflare Pages",
        href: `${baseDir}/deploy/cloudflare`,
      },
    ],
  },
];

const referenceNavigation = [
  {
    title: "Inngest Client",
    links: [
      {
        title: "Create the client",
        href: `${baseDir}/reference/client/create`,
      },
    ],
  },
  {
    title: "Functions",
    links: [
      {
        title: "Create function",
        href: `${baseDir}/reference/functions/create`,
      },
      {
        title: "Define steps (step.run)",
        href: `${baseDir}/reference/functions/step-run`,
      },
      {
        title: "Sleep",
        href: `${baseDir}/reference/functions/step-sleep`,
      },
      {
        title: "Sleep until a time",
        href: `${baseDir}/reference/functions/step-sleep-until`,
      },
      {
        title: "Wait for additional events",
        href: `${baseDir}/reference/functions/step-wait-for-event`,
      },
      {
        title: "Sending events from functions",
        href: `${baseDir}/reference/functions/step-send-event`,
      },
      {
        title: "Error handling & retries",
        href: `${baseDir}/functions/retries`,
        // href: `${baseDir}/reference/functions/error-handling`,
      },
      {
        title: "Handling failures",
        href: `${baseDir}/reference/functions/handling-failures`,
      },
      {
        title: "Cancel running functions",
        href: `${baseDir}/functions/cancellation`,
        // href: `${baseDir}/reference/functions/cancel-running-functions`,
      },
      {
        title: "Concurrency",
        href: `${baseDir}/functions/concurrency`,
        // href: `${baseDir}/reference/functions/concurrency`,
      },
      // {
      //   title: "Logging",
      //   href: `${baseDir}/reference/functions/logging`,
      // },
    ],
  },
  {
    title: "Events",
    links: [
      {
        title: "Send",
        href: `${baseDir}/reference/events/send`,
      },
    ],
  },
  {
    title: "Serve",
    links: [
      // {
      //   title: "Framework handlers",
      //   href: `${baseDir}/sdk/serve`,
      // },
      {
        title: "Configuration",
        href: `${baseDir}/reference/serve`,
      },
      { title: "Streaming", href: `${baseDir}/streaming` },
    ],
  },
  {
    title: "Using the SDK",
    links: [
      {
        title: "Using TypeScript",
        href: `${baseDir}/typescript`,
      },
      { title: "Migrating to v1", href: `${baseDir}/sdk/v1-migration` },
    ],
  },
];

export const headerLinks = [
  {
    title: "Docs",
    href: baseDir,
  },
  {
    title: "Patterns",
    href: "/patterns?ref=docs",
  },
];

export function Navigation(props) {
  return (
    <nav {...props}>
      <ul role="list">
        <li className="mt-6 mb-4 flex gap-2 items-center text-base font-semibold text-slate-900 dark:text-white">
          <span className="p-0.5 bg-indigo-500 rounded-sm">
            <IconGuide />
          </span>
          Guides
        </li>
        {/* <li>
          <Button
            href="/ai-personalized-documentation?ref=docs"
            variant="secondary"
            className="w-full mb-6 xl:hidden"
          >
            ✨ Create AI-Personalized Docs
          </Button>
        </li> */}
        {headerLinks.map((link) => (
          <TopLevelNavItem key={link.title} href={link.href}>
            {link.title}
          </TopLevelNavItem>
        ))}
        {navigation.map((group, groupIndex) => (
          <NavigationGroup
            key={group.title}
            group={group}
            className={groupIndex === 0 && "lg:mt-0"}
          />
        ))}
        <li className="mt-6 mb-4 flex gap-2 items-center text-base font-semibold text-slate-900 dark:text-white">
          <span className="p-0.5 bg-blue-500 rounded-sm">
            <IconDocs />
          </span>
          Reference
        </li>
        {referenceNavigation.map((group, groupIndex) => (
          <NavigationGroup
            key={group.title}
            group={group}
            className={groupIndex === 0 && "lg:mt-0"}
          />
        ))}
        <li className="sticky bottom-0 z-10 mt-6 sm:hidden gap-2 flex dark:bg-slate-900">
          <Button
            href={process.env.NEXT_PUBLIC_LOGIN_URL}
            variant="secondary"
            className="w-full"
          >
            Log in
          </Button>
          <Button
            href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=docs-mobile-nav`}
            variant="primary"
            arrow="right"
            className="w-full"
          >
            Sign up
          </Button>
        </li>
      </ul>
    </nav>
  );
}
