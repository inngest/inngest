import { TextCell } from '@inngest/components/Table';

import { CellPadding } from './CellPadding';
import { getFormattedJSONObjectOrArrayString } from './json';

interface JSONAwareTextCellProps {
  children: string;
}

export function JSONAwareTextCell({ children }: JSONAwareTextCellProps) {
  const formattedJSON = getFormattedJSONObjectOrArrayString(children);

  if (formattedJSON === null) {
    return (
      <CellPadding>
        <TextCell>{children}</TextCell>
      </CellPadding>
    );
  }

  return (
    <CellPadding>
      <div className="text-basis text-sm font-medium">
        <pre className="m-0 max-h-[150px] max-w-[350px] overflow-x-auto overflow-y-auto whitespace-pre break-all">
          {formattedJSON}
        </pre>
      </div>
    </CellPadding>
  );
}
