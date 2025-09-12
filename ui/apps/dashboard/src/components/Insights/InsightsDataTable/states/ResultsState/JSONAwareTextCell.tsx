import { TextCell } from '@inngest/components/Table';

import { getFormattedJSONObjectOrArrayString } from './json';

interface JSONAwareTextCellProps {
  children: string;
}

export function JSONAwareTextCell({ children }: JSONAwareTextCellProps) {
  const formattedJSON = getFormattedJSONObjectOrArrayString(children);

  if (formattedJSON === null) return <TextCell>{children}</TextCell>;

  return (
    <div className="text-basis text-sm font-medium">
      <pre
        tabIndex={-1}
        className="m-0 max-h-[150px] w-full max-w-none overflow-hidden whitespace-pre break-all focus:outline-none group-focus-within:overflow-auto"
      >
        {formattedJSON}
      </pre>
    </div>
  );
}
