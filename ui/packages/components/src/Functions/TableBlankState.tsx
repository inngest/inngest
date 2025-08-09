import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';

import { InlineCode } from '../Code';

type TableBlankStateProps = {
  actions: React.ReactNode;
  title?: string;
};

export default function TableBlankState({ actions, title }: TableBlankStateProps) {
  return (
    <div className="text-basis mt-36 flex flex-col items-center justify-center gap-5">
      <div className="bg-canvasSubtle text-light rounded-md p-3 ">
        <FunctionsIcon className="h-7 w-7" />
      </div>
      <div className="text-center">
        <p className="mb-1.5 text-xl">{title || 'No functions found'}</p>
        <p className="text-subtle max-w-md text-sm">
          To create functions, you will use <InlineCode>inngest.createFunction()</InlineCode>.
        </p>
      </div>
      <div className="flex items-center gap-3">{actions}</div>
    </div>
  );
}
