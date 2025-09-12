import { TextCell } from '@inngest/components/Table';
import { cn } from '@inngest/components/utils/classNames';

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
    <div className={cn('text-basis text-sm font-medium')}>
      <pre className="max-h-[150px] max-w-[350px] overflow-x-auto overflow-y-auto whitespace-pre">
        {formattedJSON}
      </pre>
    </div>
  );
}
