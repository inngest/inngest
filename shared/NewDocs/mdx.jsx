import Link from "next/link";
import clsx from "clsx";

import { Heading } from "./Heading";

export const a = Link;
export { Button } from "../Button";
export { CodeGroup, Code as code, Pre as pre } from "./Code";

export const h2 = function H2(props) {
  return <Heading level={2} {...props} />;
};

function InfoIcon(props) {
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true" {...props}>
      <circle cx="8" cy="8" r="8" strokeWidth="0" />
      <path
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
        d="M6.75 7.75h1.5v3.5"
      />
      <circle cx="8" cy="4" r=".5" fill="none" />
    </svg>
  );
}

export function Note({ children }) {
  return (
    <div className="my-6 flex gap-2.5 rounded-xl border border-indigo-500/20 bg-indigo-50/50 p-4 leading-6 text-indigo-900 dark:border-indigo-500/30 dark:bg-indigo-500/5 dark:text-indigo-200 dark:[--tw-prose-links:theme(colors.white)] dark:[--tw-prose-links-hover:theme(colors.indigo.300)]">
      <InfoIcon className="mt-1 h-4 w-4 flex-none fill-indigo-500 stroke-white dark:fill-indigo-200/20 dark:stroke-indigo-200" />
      <div className="[&>:first-child]:mt-0 [&>:last-child]:mb-0">
        {children}
      </div>
    </div>
  );
}

export function Callout({ children }) {
  return (
    <div className="border border-transparent dark:border-indigo-600/20 text-indigo-600 dark:text-indigo-200 bg-indigo-600/10 rounded-lg p-6  [&>:first-child]:mt-0 [&>:last-child]:mb-0">
      {children}
    </div>
  );
}

export function ButtonCol({ children }) {
  return (
    <div className="flex flex-col lg:flex-row gap-4 w-full justify-between">
      {children}
    </div>
  );
}

export function ButtonDeploy({ label, type, href }) {
  return (
    <a
      href={href}
      className=" bg-indigo-200 hover:bg-indigo-300/60 hover:shadow-md dark:bg-indigo-900/20 block rounded-lg p-4 group/deploy dark:hover:bg-indigo-800/40 dark:borde dark:border-indigo-500/20 no-underline transition-all"
    >
      <img
        src={`/assets/docs/logos/${type}.svg`}
        className="w-24 mt-0 mb-2 pt-2 pb-1"
      />
      <span className="text-slate-700 dark:text-slate-100 text-sm">
        {label}
      </span>
    </a>
  );
}

export function Row({ children }) {
  return (
    <div className="grid grid-cols-1 items-start gap-x-16 gap-y-10 xl:max-w-none xl:grid-cols-2">
      {children}
    </div>
  );
}

export function Col({ children, sticky = false }) {
  return (
    <div
      className={clsx(
        "[&>:first-child]:mt-0 [&>:last-child]:mb-0",
        sticky && "xl:sticky xl:top-24"
      )}
    >
      {children}
    </div>
  );
}

export function Properties({ children }) {
  return (
    <div className="my-6">
      <ul
        role="list"
        className="m-0 max-w-[calc(theme(maxWidth.lg)-theme(spacing.8))] list-none divide-y divide-slate-900/5 p-0 dark:divide-white/5"
      >
        {children}
      </ul>
    </div>
  );
}

export function Property({ name, type, children }) {
  return (
    <li className="m-0 px-0 py-4 first:pt-0 last:pb-0">
      <dl className="m-0 flex flex-wrap items-center gap-x-3 gap-y-2">
        <dt className="sr-only">Name</dt>
        <dd>
          <code>{name}</code>
        </dd>
        <dt className="sr-only">Type</dt>
        <dd className="font-mono text-xs text-slate-400 dark:text-slate-500">
          {type}
        </dd>
        <dt className="sr-only">Description</dt>
        <dd className="w-full flex-none [&>:first-child]:mt-0 [&>:last-child]:mb-0">
          {children}
        </dd>
      </dl>
    </li>
  );
}
