import { Disclosure, Transition } from "@headlessui/react";
import classNames from "@/utils/classnames";
import {
  IconStatusCompleted,
  IconChevron,
  IconCheckCircle,
  IconExclamationTriangle,
  IconTrash,
} from "@/icons";
import Button from "../Button";

type AppCardProps = {
  app: {
    id: string;
    name: string;
    url: string;
    functionCount: number;
    sdkVersion: string;
    status: string;
    automaticallyAdded: boolean;
  };
};

const AppHeader = ({ status, functionCount, automaticallyAdded }) => {
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
      <div className="flex items-center gap-2">
        {headerIcon}
        {headerLabel}
      </div>
      {!automaticallyAdded && (
        <Button kind="text" icon={<IconTrash />} btnAction={() => {}} />
      )}
    </header>
  );
};

export default function AppCard({ app }: AppCardProps) {
  return (
    <div className="bg-slate-800/30">
      <AppHeader
        status={app.status}
        functionCount={app.functionCount}
        automaticallyAdded={app.automaticallyAdded}
      />
      <div className="border border-slate-700/30 rounded-b-md divide-y divide-slate-700/30">
        <div className="flex items-center justify-between px-6 py-4 ">
          <p className=" text-base text-white">{app.name}</p>
          {!app.automaticallyAdded && (
            <span className="text-xs border rounded-md border-slate-800 py-1.5 px-2.5 text-slate-300">
              Auto Detected
            </span>
          )}
        </div>

        <Disclosure
          as="div"
          className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800"
        >
          <Disclosure.Button className="flex items-center text-white justify-between p-4 w-full">
            <div className="flex items-center gap-3">
              {<IconStatusCompleted />}Connected to server
            </div>
            <div className="flex items-center gap-4">
              <p className="text-slate-300">{app.url}</p>
              <IconChevron className="transform-90 text-slate-500"/>
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
            <Disclosure.Panel className="text-gray-500">
              Something
            </Disclosure.Panel>
          </Transition>
        </Disclosure>
        <div className="flex items-center text-white justify-between p-4">
          <div className="flex items-center gap-3">
            {<IconStatusCompleted />}
            {app.functionCount} Functions registered
          </div>
          <button className="text-indigo-400 flex items-center gap-2">
            View Functions
            <IconChevron className="-rotate-90" />
          </button>
        </div>
      </div>
    </div>
  );
}
