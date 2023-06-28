import { Disclosure, Transition } from "@headlessui/react";
import { IconStatusCompleted } from "@/icons";

type AppCardProps = {
  app: {
    id: string;
    name: string;
    url: string;
    functionCount: number;
    sdkVersion: string;
    status: string;
    manuallyAdded: boolean;
  };
};

export default function AppCard({ app }: AppCardProps) {
  return (
    <div className="bg-slate-800/30">
      <header className="bg-teal-600 text-white rounded-t-md px-6 py-2.5 capitalize flex gap-2 items-center">
        <IconStatusCompleted />
        {app.status}
      </header>
      <div className="border border-slate-700/30 rounded-b-md divide-y divide-slate-700/30">
        <p className="px-6 py-4 text-base text-white">{app.name}</p>
        <Disclosure
          as="div"
          className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800"
        >
          <Disclosure.Button className="flex items-center text-white justify-between p-4 w-full">
            <div className="flex items-center gap-3">
              {<IconStatusCompleted />}Connected to server
            </div>
            <p className="text-slate-300">{app.url}</p>
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
          <button className="text-indigo-400">View Functions</button>
        </div>
      </div>
    </div>
  );
}
