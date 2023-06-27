import Link, { LinkProps } from "next/link";
import clsx from "clsx";

import { Heading } from "./Heading";
import React, { useState } from "react";
import { ChevronDown, ChevronUp } from "react-feather";

// export const a: React.FunctionComponent<LinkProps> = (props) => (
//   <Link {...props} />
// );

export const a: React.FunctionComponent<
  React.AnchorHTMLAttributes<HTMLAnchorElement>
> = ({ children, href, target, rel, download }) => (
  <Link href={href} target={target} rel={rel} download={download}>
    {children}
  </Link>
);

export { Button } from "../Button";
export {
  CodeGroup,
  Code as code,
  Pre as pre,
  GuideSelector,
  GuideSection,
} from "./Code";

export const h2: React.FC<any> = function H2(props) {
  return <Heading level={2} {...props} />;
};
export const h3: React.FC<any> = function H2(props) {
  return <Heading level={3} {...props} />;
};
export const h4: React.FC<any> = function H2(props) {
  return <Heading level={3} {...props} />;
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

function VercelLogo() {
  return (
    <svg
      viewBox="0 0 4438 1000"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-20 m-auto mb-3.5"
    >
      <path
        d="M2223.75 250C2051.25 250 1926.87 362.5 1926.87 531.25C1926.87 700 2066.72 812.5 2239.38 812.5C2343.59 812.5 2435.47 771.25 2492.34 701.719L2372.81 632.656C2341.25 667.188 2293.28 687.344 2239.38 687.344C2164.53 687.344 2100.94 648.281 2077.34 585.781H2515.16C2518.59 568.281 2520.63 550.156 2520.63 531.094C2520.63 362.5 2396.41 250 2223.75 250ZM2076.09 476.562C2095.62 414.219 2149.06 375 2223.75 375C2298.59 375 2352.03 414.219 2371.41 476.562H2076.09ZM2040.78 78.125L1607.81 828.125L1174.69 78.125H1337.03L1607.66 546.875L1878.28 78.125H2040.78ZM577.344 0L1154.69 1000H0L577.344 0ZM3148.75 531.25C3148.75 625 3210 687.5 3305 687.5C3369.38 687.5 3417.66 658.281 3442.5 610.625L3562.5 679.844C3512.81 762.656 3419.69 812.5 3305 812.5C3132.34 812.5 3008.13 700 3008.13 531.25C3008.13 362.5 3132.5 250 3305 250C3419.69 250 3512.66 299.844 3562.5 382.656L3442.5 451.875C3417.66 404.219 3369.38 375 3305 375C3210.16 375 3148.75 437.5 3148.75 531.25ZM4437.5 78.125V796.875H4296.88V78.125H4437.5ZM3906.25 250C3733.75 250 3609.38 362.5 3609.38 531.25C3609.38 700 3749.38 812.5 3921.88 812.5C4026.09 812.5 4117.97 771.25 4174.84 701.719L4055.31 632.656C4023.75 667.188 3975.78 687.344 3921.88 687.344C3847.03 687.344 3783.44 648.281 3759.84 585.781H4197.66C4201.09 568.281 4203.12 550.156 4203.12 531.094C4203.12 362.5 4078.91 250 3906.25 250ZM3758.59 476.562C3778.13 414.219 3831.41 375 3906.25 375C3981.09 375 4034.53 414.219 4053.91 476.562H3758.59ZM2961.25 265.625V417.031C2945.63 412.5 2929.06 409.375 2911.25 409.375C2820.47 409.375 2755 471.875 2755 565.625V796.875H2614.38V265.625H2755V409.375C2755 330 2847.34 265.625 2961.25 265.625Z"
        fill="currentColor"
      />
    </svg>
  );
}

function NetlifyLogo() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 147 40"
      className="w-20 m-auto mb-3"
    >
      <g fill="currentColor" fillRule="evenodd">
        <path d="M53.37 12.978l.123 2.198c1.403-1.7 3.245-2.55 5.525-2.55 3.951 0 5.962 2.268 6.032 6.804v12.568H60.79V19.676c0-1.207-.26-2.1-.78-2.681-.52-.58-1.371-.87-2.552-.87-1.719 0-3 .78-3.84 2.338v13.535h-4.262v-19.02h4.016zM77.748 32.35c-2.7 0-4.89-.852-6.567-2.557-1.678-1.705-2.517-3.976-2.517-6.812v-.527c0-1.898.365-3.595 1.096-5.089.73-1.494 1.757-2.657 3.078-3.49 1.321-.831 2.794-1.247 4.42-1.247 2.583 0 4.58.826 5.988 2.478 1.41 1.653 2.114 3.99 2.114 7.014v1.723h-12.4c.13 1.57.652 2.812 1.57 3.726.918.914 2.073 1.371 3.464 1.371 1.952 0 3.542-.79 4.77-2.373l2.297 2.198c-.76 1.136-1.774 2.018-3.042 2.645-1.269.627-2.692.94-4.27.94zm-.508-16.294c-1.17 0-2.113.41-2.832 1.23-.72.82-1.178 1.963-1.377 3.428h8.12v-.317c-.094-1.43-.474-2.51-1.14-3.243-.667-.732-1.59-1.098-2.771-1.098zm16.765-7.7v4.623h3.35v3.164h-3.35V26.76c0 .726.144 1.25.43 1.573.286.322.798.483 1.535.483a6.55 6.55 0 0 0 1.49-.176v3.305c-.97.27-1.905.404-2.806.404-3.273 0-4.91-1.81-4.91-5.431V16.142H86.62v-3.164h3.122V8.355h4.261zm11.137 23.643h-4.262v-27h4.262v27zm9.172 0h-4.262v-19.02h4.262v19.02zm-4.525-23.96c0-.655.207-1.2.622-1.634.416-.433 1.009-.65 1.78-.65.772 0 1.368.217 1.79.65.42.434.63.979.63 1.635 0 .644-.21 1.18-.63 1.608-.422.428-1.018.642-1.79.642-.771 0-1.364-.214-1.78-.642-.415-.427-.622-.964-.622-1.608zm10.663 23.96V16.142h-2.894v-3.164h2.894v-1.74c0-2.11.584-3.738 1.753-4.887 1.17-1.148 2.806-1.722 4.91-1.722.749 0 1.544.105 2.386.316l-.105 3.34a8.375 8.375 0 0 0-1.631-.14c-2.035 0-3.052 1.048-3.052 3.146v1.687h3.858v3.164h-3.858v15.856h-4.261zm17.87-6.117l3.858-12.903h4.542l-7.54 21.903c-1.158 3.199-3.122 4.799-5.893 4.799-.62 0-1.304-.106-2.052-.317v-3.305l.807.053c1.075 0 1.885-.196 2.429-.589.543-.392.973-1.051 1.289-1.977l.613-1.635-6.664-18.932h4.595l4.016 12.903z" />
        <path
          fillRule="nonzero"
          d="M27.887 14.135l-.014-.006c-.008-.003-.016-.006-.023-.013a.11.11 0 0 1-.028-.093l.773-4.726 3.625 3.626-3.77 1.604a.083.083 0 0 1-.033.006h-.015c-.005-.003-.01-.007-.02-.017a1.716 1.716 0 0 0-.495-.381zm5.258-.288l3.876 3.876c.805.806 1.208 1.208 1.355 1.674.022.069.04.138.054.209l-9.263-3.923a.728.728 0 0 0-.015-.006c-.037-.015-.08-.032-.08-.07 0-.038.044-.056.081-.071l.012-.005 3.98-1.684zm5.127 7.003c-.2.376-.59.766-1.25 1.427l-4.37 4.369L27 25.469l-.03-.006c-.05-.008-.103-.017-.103-.062a1.706 1.706 0 0 0-.655-1.193c-.023-.023-.017-.059-.01-.092 0-.005 0-.01.002-.014l1.063-6.526.004-.022c.006-.05.015-.108.06-.108a1.73 1.73 0 0 0 1.16-.665c.009-.01.015-.021.027-.027.032-.015.07 0 .103.014l9.65 4.082zm-6.625 6.801l-7.186 7.186 1.23-7.56.002-.01c.001-.01.003-.02.006-.029.01-.024.036-.034.061-.044l.012-.005a1.85 1.85 0 0 0 .695-.517c.024-.028.053-.055.09-.06a.09.09 0 0 1 .029 0l5.06 1.04zm-8.707 8.707l-.81.81-8.955-12.942a.424.424 0 0 0-.01-.014c-.014-.019-.029-.038-.026-.06 0-.016.011-.03.022-.042l.01-.013c.027-.04.05-.08.075-.123l.02-.035.003-.003c.014-.024.027-.047.051-.06.021-.01.05-.006.073-.001l9.921 2.046a.164.164 0 0 1 .076.033c.013.013.016.027.019.043a1.757 1.757 0 0 0 1.028 1.175c.028.014.016.045.003.078a.238.238 0 0 0-.015.045c-.125.76-1.197 7.298-1.485 9.063zm-1.692 1.691c-.597.591-.949.904-1.347 1.03a2 2 0 0 1-1.206 0c-.466-.148-.869-.55-1.674-1.356L8.028 28.73l2.349-3.643c.011-.018.022-.034.04-.047.025-.018.061-.01.091 0a2.434 2.434 0 0 0 1.638-.083c.027-.01.054-.017.075.002a.19.19 0 0 1 .028.032l8.999 13.058zM7.16 27.863L5.098 25.8l4.074-1.738a.084.084 0 0 1 .033-.007c.034 0 .054.034.072.065a2.91 2.91 0 0 0 .13.184l.013.016c.012.017.004.034-.008.05l-2.25 3.493zm-2.976-2.976l-2.61-2.61c-.444-.444-.766-.766-.99-1.043l7.936 1.646a.84.84 0 0 0 .03.005c.049.008.103.017.103.063 0 .05-.059.073-.109.092l-.023.01-4.337 1.837zM.13 19.892a2 2 0 0 1 .09-.495c.148-.466.55-.868 1.356-1.674l3.34-3.34a2175.525 2175.525 0 0 0 4.626 6.687c.027.036.057.076.026.106-.146.161-.292.337-.395.528a.16.16 0 0 1-.05.062c-.013.008-.027.005-.042.002h-.002L.129 19.891zm5.68-6.403l4.49-4.491c.423.185 1.96.834 3.333 1.414 1.04.44 1.988.84 2.286.97.03.012.057.024.07.054.008.018.004.041 0 .06a2.003 2.003 0 0 0 .523 1.828c.03.03 0 .073-.026.11l-.014.021-4.56 7.063c-.012.02-.023.037-.043.05-.024.015-.058.008-.086.001a2.274 2.274 0 0 0-.543-.074c-.164 0-.342.03-.522.063h-.001c-.02.003-.038.007-.054-.005a.21.21 0 0 1-.045-.051l-4.808-7.013zm5.398-5.398l5.814-5.814c.805-.805 1.208-1.208 1.674-1.355a2 2 0 0 1 1.206 0c.466.147.869.55 1.674 1.355l1.26 1.26L18.7 9.94a.155.155 0 0 1-.041.048c-.025.017-.06.01-.09 0a2.097 2.097 0 0 0-1.92.37c-.027.028-.067.012-.101-.003-.54-.235-4.74-2.01-5.341-2.265zm12.506-3.676l3.818 3.818-.92 5.698v.015a.135.135 0 0 1-.008.038c-.01.02-.03.024-.05.03a1.83 1.83 0 0 0-.548.273.154.154 0 0 0-.02.017c-.011.012-.022.023-.04.025a.114.114 0 0 1-.043-.007l-5.818-2.472-.011-.005c-.037-.015-.081-.033-.081-.071a2.198 2.198 0 0 0-.31-.915c-.028-.046-.059-.094-.035-.141l4.066-6.303zM19.78 13.02l5.454 2.31c.03.014.063.027.076.058a.106.106 0 0 1 0 .057c-.016.08-.03.171-.03.263v.153c0 .038-.039.054-.075.069l-.011.004c-.864.369-12.13 5.173-12.147 5.173-.017 0-.035 0-.052-.017-.03-.03 0-.072.027-.11a.76.76 0 0 0 .014-.02l4.482-6.94.008-.012c.026-.042.056-.089.104-.089l.045.007c.102.014.192.027.283.027.68 0 1.31-.331 1.69-.897a.16.16 0 0 1 .034-.04c.027-.02.067-.01.098.004zm-6.246 9.185l12.28-5.237s.018 0 .035.017c.067.067.124.112.179.154l.027.017c.025.014.05.03.052.056 0 .01 0 .016-.002.025L25.054 23.7l-.004.026c-.007.05-.014.107-.061.107a1.729 1.729 0 0 0-1.373.847l-.005.008c-.014.023-.027.045-.05.057-.021.01-.048.006-.07.001l-9.793-2.02c-.01-.002-.152-.519-.163-.52z"
        />
      </g>
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

export function Callout({
  variant = "default",
  children,
}: {
  variant: "default" | "info" | "warning";
  children: React.ReactNode;
}) {
  return (
    <div
      className={clsx(
        "my-6 border border-transparent rounded-lg p-6 mt- [&>:first-child]:mt-0 [&>:last-child]:mb-0",
        variant === "default" &&
          "dark:border-indigo-600/20 text-indigo-600 dark:text-indigo-200 bg-indigo-600/10",
        variant === "info" &&
          "dark:border-sky-600/20 text-sky-600 dark:text-sky-100 bg-sky-300/10",
        variant === "warning" &&
          "dark:border-amber-700/20 text-amber-900 dark:text-amber-50 bg-amber-300/10"
      )}
    >
      {children}
    </div>
  );
}

export function ButtonCol({ children }) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-2 w-full justify-between">
      {children}
    </div>
  );
}

export function ButtonDeploy({ label, type, href }) {
  let logoType;

  // change logo based on type
  switch (type) {
    case "vercel":
      logoType = <VercelLogo />;
      break;
    case "netlify":
      logoType = <NetlifyLogo />;
      break;
    default:
      logoType = <VercelLogo />;
  }

  return (
    <a
      href={href}
      target="_blank"
      className="block text-slate-800 dark:text-slate-200 cursor-pointer bg-slate-200 hover:bg-indigo-300/60 dark:bg-slate-800/40 rounded-lg py-8 px-6 group/deploy dark:hover:bg-indigo-800/40 no-underline transition-all"
    >
      {logoType}
      <span className="text-slate-700 dark:text-slate-100 text-sm text-center w-full block">
        {label}
      </span>
    </a>
  );
}

export function Row({ children }) {
  return (
    <div className="grid grid-cols-1 items-start gap-x-16 gap-y-10 xl:max-w-none xl:grid-cols-2 my-6 [&>:first-child]:mt-0 [&>:last-child]:mb-0">
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

export function Properties({
  nested = false,
  collapse = false,
  name = "Properties",
  children,
}: {
  nested?: boolean;
  collapse?: boolean;
  name?: string;
  children: React.ReactElement;
}) {
  const [isCollapsed, setCollapsed] = useState<boolean>(collapse);
  return (
    <div
      className={clsx(
        "my-6",
        nested &&
          "-mt-3 pt-2 pb-3 border border-slate-900/5 dark:border-white/5 rounded-md",
        collapse && isCollapsed && "pb-0"
      )}
    >
      {nested && (
        <div
          className={clsx(
            "px-3 pb-2 mb-3 border-b border-slate-900/5 text-xs font-semibold",
            collapse && isCollapsed && "mb-0 border-b-0"
          )}
        >
          {collapse ? (
            <button
              className="flex gap-1 items-center hover:text-indigo-800	"
              onClick={() => {
                setCollapsed(!isCollapsed);
              }}
            >
              {`Show nested ${name.toLowerCase()} `}
              {isCollapsed ? (
                <ChevronDown className="h-3" />
              ) : (
                <ChevronUp className="h-3" />
              )}
            </button>
          ) : (
            name
          )}
        </div>
      )}
      <ul
        role="list"
        className={clsx(
          "m-0 max-w-[calc(theme(maxWidth.3xl)-theme(spacing.8))] list-none divide-y divide-slate-900/5 p-0 dark:divide-white/5",
          nested && "px-3",
          collapse && isCollapsed && "hidden"
        )}
      >
        {children}
      </ul>
    </div>
  );
}

export function Property({
  name,
  type,
  required = false,
  attribute = false,
  version,
  children,
}: {
  name: string;
  type: string;
  /** Is the property a required argument? */
  required?: boolean;
  /** Is the property an attribute part of an object? (required/optional will be hidden) */
  attribute?: boolean;
  /** The version at which this property is available, e.g. "v2.0.0+" */
  version?: string;
  children: React.ReactElement;
}) {
  return (
    <li id={name} className="m-0 px-0 py-4 first:pt-0 last:pb-0 scroll-mt-24">
      <dl className="m-0 flex flex-wrap items-center gap-x-3 gap-y-2">
        <dt className="sr-only">Name</dt>
        <dd>
          <code>{name}</code>
        </dd>
        <dt className="sr-only">Type</dt>
        <dd className="font-mono font-medium text-xs text-slate-500 dark:text-slate-400">
          {type}
        </dd>
        {!attribute && (
          <>
            <dt className="sr-only">Required</dt>
            <dd
              className={clsx(
                "font-mono font-medium text-xs",
                required
                  ? "text-amber-600 dark:text-amber-400"
                  : "text-slate-500 dark:text-slate-400"
              )}
            >
              {required ? "required" : "optional"}
            </dd>
          </>
        )}
        {version ? (
          <div>
            <dt className="sr-only">Version</dt>
            <VersionBadge version={version as `v${string}`} />
          </div>
        ) : null}
        <dt className="sr-only">Description</dt>
        <dd className="w-full text-sm flex-none [&>:first-child]:mt-0 [&>:last-child]:mb-0">
          {children}
        </dd>
      </dl>
    </li>
  );
}

export function VersionBadge({ version }: { version: `v${string}` }) {
  return (
    <div className="inline-flex items-center px-3 py-0.5 rounded-full text-xs font-medium leading-4 bg-slate-200 text-slate-800 dark:bg-slate-800 dark:text-slate-200">
      <span>{version}</span>
    </div>
  );
}
