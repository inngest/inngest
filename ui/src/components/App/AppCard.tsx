import { Disclosure, Transition } from "@headlessui/react";
import classNames from "@/utils/classnames";
import {
  IconAppStatusCompleted,
  IconAppStatusFailed,
  IconChevron,
  IconCheckCircle,
  IconExclamationTriangle,
  IconSpinner,
  IconArrowTopRightOnSquare,
  IconAppStatusDefault,
} from "@/icons";

type AppCardProps = {
  app: {
    id: string;
    name: string;
    url: string;
    functionCount: number;
    sdkVersion: string;
    status: string;
    automaticallyAdded: boolean;
    connecting?: boolean;
  };
};

const AppHeader = ({ status, functionCount, sdkVersion }) => {
  let headerColor, headerLabel, headerIcon;

  if (status !== "connected") {
    headerColor = "bg-rose-600/50";
    headerLabel = "No Connection";
    headerIcon = <IconExclamationTriangle />;
  } else if (functionCount < 1) {
    headerColor = "bg-orange-400/70";
    headerLabel = "No Functions Found";
    headerIcon = <IconExclamationTriangle />;
  } else {
    headerColor = "bg-teal-400/50";
    headerLabel = "Connected";
    headerIcon = <IconCheckCircle />;
  }

  return (
    <header
      className={classNames(
        headerColor,
        `text-white rounded-t-md px-6 py-2.5 capitalize flex gap-2 items-center justify-between`
      )}
    >
      <div className="flex items-center gap-2 leading-7">
        {headerIcon}
        {headerLabel}
      </div>
      {sdkVersion && (
        <span className="text-xs leading-3 border rounded-md border-white/20 box-border py-1.5 px-2 text-slate-300">
          SDK {sdkVersion}
        </span>
      )}
    </header>
  );
};

export default function AppCard({ app }: AppCardProps) {
  return (
    <div>
      <AppHeader
        status={app.status}
        functionCount={app.functionCount}
        sdkVersion={app.sdkVersion}
      />
      <div className="border border-slate-700/30 rounded-b-md divide-y divide-slate-700/30 bg-slate-800/30">
        {app.connecting ? (
          <div className="p-4 pr-6 flex items-center gap-2">
            <IconSpinner className="fill-sky-400 text-slate-800" />
            <p className="text-slate-400 text-lg font-light">Connecting...</p>
          </div>
        ) : (
          <div className="flex items-center justify-between px-6 py-4 ">
            <p className=" text-lg text-white">{app.name}</p>
            {app.automaticallyAdded && (
              <span className="text-xs leading-3 border rounded-md border-slate-800 box-border py-1.5 px-2 text-slate-300">
                Auto Detected
              </span>
            )}
          </div>
        )}

        <Disclosure
          as="div"
          className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800"
        >
          <Disclosure.Button className="flex items-center text-white justify-between p-4 pr-6 w-full">
            <div className="flex items-center gap-3 text-base">
              {app.status === "connected" ? (
                <>{<IconAppStatusCompleted />}Connected to server</>
              ) : (
                <>{<IconAppStatusFailed />}No connection to server</>
              )}
            </div>
            <div className="flex items-center gap-4">
              <p className="text-slate-300 ui-open:hidden">{app.url}</p>
              <IconChevron className="ui-open:-rotate-180 transform-90 text-slate-500" />
            </div>
          </Disclosure.Button>
          <Transition
            enter="transition-opacity duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="transition-opacity duration-300"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Disclosure.Panel className="text-gray-500 pl-10 pr-6 pb-4 ">
              {app.status !== "connected" && (
                <p className="pb-4 text-slate-400">
                  The Inngest Dev Server can’t find your application. Ensure
                  your full URL is correct, including the correct port. Inngest
                  automatically scans{" "}
                  <span className="text-white">multiple ports</span> by default.
                </p>
              )}
              <div className="flex items-center justify-between pb-4">
                <div>
                  <p className="text-sm font-semibold text-white">App URL</p>
                  <p className="text-slate-400">The URL of your application</p>
                </div>
                <input
                  className="min-w-[50%] bg-slate-800 rounded-md text-slate-300 py-2 px-4"
                  value={app.url}
                  readOnly={app.automaticallyAdded}
                />
              </div>
              <a className="text-indigo-400 flex items-center gap-2">
                Connecting to the Dev Server
                <IconArrowTopRightOnSquare />
              </a>
            </Disclosure.Panel>
          </Transition>
        </Disclosure>

        <Disclosure
          as="div"
          className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800"
        >
          <Disclosure.Button className="flex items-center text-white justify-between p-4 pr-6 w-full">
            <div className="flex items-center gap-3 text-base">
              {app.status === "connected" && app.functionCount > 0 ? (
                <>
                  {<IconAppStatusCompleted />}
                  {app.functionCount} Functions registered
                </>
              ) : app.status !== "connected" ? (
                <>{<IconAppStatusDefault />}No Functions Found</>
              ) : (
                <>{<IconAppStatusFailed />}No Functions Found</>
              )}
            </div>
            <div className="flex items-center gap-4">
              {app.status === "connected" && app.functionCount > 0 ? (
                <>
                  <button className="text-indigo-400 flex items-center gap-2">
                    View Functions
                    <IconChevron className="-rotate-90" />
                  </button>
                </>
              ) : (
                <IconChevron className="ui-open:-rotate-180 transform-90 text-slate-500" />
              )}
            </div>
          </Disclosure.Button>
          {(app.status !== "connected" || app.functionCount < 1) && (
            <Transition
              enter="transition-opacity duration-300"
              enterFrom="opacity-0"
              enterTo="opacity-100"
              leave="transition-opacity duration-300"
              leaveFrom="opacity-100"
              leaveTo="opacity-0"
            >
              <Disclosure.Panel className="text-gray-500 pl-10 pr-6 pb-4 ">
                <p className="pb-4 text-slate-400">
                  There are currently no functions registered at this url.
                  Ensure you have created a function and are exporting it
                  correctly from your serve command.
                </p>
                <div className="flex items-center justify-between p-4 mb-4 bg-slate-950 rounded-md">
                  <code className="text-slate-300">
                    serve(client, [list_of_fns]);
                  </code>
                </div>
                <a className="text-indigo-400 flex items-center gap-2">
                  Creating Functions
                  <IconArrowTopRightOnSquare />
                </a>
              </Disclosure.Panel>
            </Transition>
          )}
        </Disclosure>
        {!app.automaticallyAdded && (
          <div className="text-white p-4 pr-6">
            <button className="text-rose-400">Delete App</button>
          </div>
        )}
      </div>
    </div>
  );
}
