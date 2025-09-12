import { TextCell } from '@inngest/components/Table';

import { getFormattedJSONObjectOrArrayString } from './json';

interface JSONAwareTextCellProps {
  children: string;
}

export function JSONAwareTextCell({ children }: JSONAwareTextCellProps) {
  const formattedJSON = getFormattedJSONObjectOrArrayString(children);
  if (formattedJSON === null) {
    return <TextCell>{children}</TextCell>;
  }

  return (
    <TextCell>
      <pre className="m-0 whitespace-pre-wrap">{formattedJSON}</pre>
    </TextCell>
  );
}
