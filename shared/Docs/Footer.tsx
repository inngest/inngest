import React, { forwardRef, Fragment, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/router";
import { Transition } from "@headlessui/react";

import { Button } from "../Button";
import { navigation } from "./Navigation";
import SocialBadges from "./SocialBadges";

function CheckIcon(props) {
  return (
    <svg viewBox="0 0 20 20" aria-hidden="true" {...props}>
      <circle cx="10" cy="10" r="10" strokeWidth="0" />
      <path
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
        d="m6.75 10.813 2.438 2.437c1.218-4.469 4.062-6.5 4.062-6.5"
      />
    </svg>
  );
}

function FeedbackButton(props) {
  return (
    <button
      type="submit"
      className="px-3 text-sm font-medium text-slate-600 transition hover:bg-slate-900/2.5 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-white"
      {...props}
    />
  );
}

const FeedbackForm = forwardRef<
  HTMLFormElement,
  { onSubmit: React.FormEventHandler }
>(function FeedbackForm({ onSubmit }, ref) {
  return (
    <form
      ref={ref}
      onSubmit={onSubmit}
      className="absolute inset-0 flex items-center justify-center gap-6 md:justify-start"
    >
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Was this page helpful?
      </p>
      <div className="group grid h-8 grid-cols-[1fr,1px,1fr] overflow-hidden rounded-full border border-slate-900/10 dark:border-white/10">
        <FeedbackButton data-response="yes">Yes</FeedbackButton>
        <div className="bg-slate-900/10 dark:bg-white/10" />
        <FeedbackButton data-response="no">No</FeedbackButton>
      </div>
    </form>
  );
});

const FeedbackThanks = forwardRef<HTMLDivElement, {}>(function FeedbackThanks(
  _props,
  ref
) {
  return (
    <div
      ref={ref}
      className="absolute inset-0 flex justify-center md:justify-start"
    >
      <div className="flex items-center gap-3 rounded-full bg-indigo-50/50 py-1 pr-3 pl-1.5 text-sm text-indigo-900 ring-1 ring-inset ring-indigo-500/20 dark:bg-indigo-500/5 dark:text-indigo-200 dark:ring-indigo-500/30">
        <CheckIcon className="h-5 w-5 flex-none fill-indigo-500 stroke-white dark:fill-indigo-200/20 dark:stroke-indigo-200" />
        Thanks for your feedback!
      </div>
    </div>
  );
});

function Feedback() {
  let [submitted, setSubmitted] = useState(false);

  function onSubmit(event) {
    event.preventDefault();

    // event.nativeEvent.submitter.dataset.response
    // => "yes" or "no"

    setSubmitted(true);
  }

  return (
    <div className="relative h-8">
      <Transition
        show={!submitted}
        as={Fragment}
        leaveFrom="opacity-100"
        leaveTo="opacity-0"
        leave="pointer-events-none duration-300"
      >
        <FeedbackForm onSubmit={onSubmit} />
      </Transition>
      <Transition
        show={submitted}
        as={Fragment}
        enterFrom="opacity-0"
        enterTo="opacity-100"
        enter="delay-150 duration-300"
      >
        <FeedbackThanks />
      </Transition>
    </div>
  );
}

function PageLink({ label, page, previous = false }) {
  return (
    <>
      <Button
        href={page.href}
        aria-label={`${label}: ${page.title}`}
        variant="secondary"
        arrow={previous ? "left" : "right"}
        size="sm"
      >
        {label}
      </Button>
      <Link
        href={page.href}
        tabIndex={-1}
        aria-hidden="true"
        className="text-base font-semibold text-slate-900 transition hover:text-slate-600 dark:text-white dark:hover:text-slate-300"
      >
        {page.title}
      </Link>
    </>
  );
}

function PageNavigation() {
  let router = useRouter();
  let allPages = navigation.flatMap((group) => group.links);
  let currentPageIndex = allPages.findIndex(
    (page) => page.href === router.pathname
  );

  if (currentPageIndex === -1) {
    return null;
  }

  let previousPage = allPages[currentPageIndex - 1];
  let nextPage = allPages[currentPageIndex + 1];

  if (!previousPage && !nextPage) {
    return null;
  }

  return (
    <div className="flex">
      {previousPage && (
        <div className="flex flex-col items-start gap-3">
          <PageLink label="Previous" page={previousPage} previous />
        </div>
      )}
      {nextPage && (
        <div className="ml-auto flex flex-col items-end gap-3">
          <PageLink label="Next" page={nextPage} />
        </div>
      )}
    </div>
  );
}

function SmallPrint() {
  return (
    <div className="flex flex-col items-center justify-between gap-5 border-t border-slate-900/5 pt-8 dark:border-white/5 sm:flex-row">
      <p className="text-xs text-slate-600 dark:text-slate-400">
        &copy; {new Date().getFullYear()} Inngest Inc. All rights reserved.
      </p>
      <SocialBadges />
    </div>
  );
}

export function Footer() {
  let router = useRouter();

  return (
    <footer className="mx-auto max-w-2xl space-y-10 pb-16 lg:max-w-5xl">
      {/* <Feedback key={router.pathname} /> */}
      <PageNavigation />
      <SmallPrint />
    </footer>
  );
}
