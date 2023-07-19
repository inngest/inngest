import { Disclosure, Transition } from '@headlessui/react';
import classNames from '@/utils/classnames';

type AppCardStepProps = {
  isExpandable?: boolean;
  lineContent: React.ReactNode;
  expandedContent?: React.ReactNode;
  isEvenStep?: boolean;
};

export default function AppCardStep({
  isExpandable = true,
  lineContent,
  expandedContent,
  isEvenStep = false,
}: AppCardStepProps) {
  const verticalLineForOddStepsclassNames = `top-[2.7rem] left-[1.844rem] h-[calc(100%-2.7rem)]`;
  const verticalLineForEvenStepsclassNames = `top-0 left-[1.844rem] h-[1.05rem]`;
  return (
    <Disclosure
      as="div"
      className="ui-open:ring-inset ui-open:ring-1 ui-open:ring-slate-800 relative"
    >
      <span
        className={classNames(
          `absolute w-px bg-slate-800`,
          isEvenStep
            ? verticalLineForEvenStepsclassNames
            : verticalLineForOddStepsclassNames
        )}
        aria-hidden="true"
      />
      <Disclosure.Button
        as={'div'}
        className="flex items-center text-white justify-between p-4 pr-6 w-full cursor-pointer"
      >
        {lineContent}
      </Disclosure.Button>
      {expandedContent && isExpandable && (
        <Transition
          enter="transition-opacity duration-200"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="transition-opacity duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Disclosure.Panel className="text-gray-500 pl-14 pr-6 pb-4 ">
            {expandedContent}
          </Disclosure.Panel>
        </Transition>
      )}
    </Disclosure>
  );
}
