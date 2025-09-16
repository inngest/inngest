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
        className="m-0 max-h-[150px] w-full max-w-none overflow-hidden whitespace-pre break-all outline-none [scrollbar-width:thin] group-focus-within:overflow-auto [&::-webkit-scrollbar]:w-1"
        tabIndex={0}
      >
        {formattedJSON}
      </pre>
    </div>
  );
}
