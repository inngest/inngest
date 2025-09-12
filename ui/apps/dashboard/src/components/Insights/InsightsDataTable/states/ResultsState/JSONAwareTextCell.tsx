import { Button } from '@inngest/components/Button/Button';
import { TextCell } from '@inngest/components/Table';
import { RiSidebarFoldLine } from '@remixicon/react';

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
      <ExpandSidebarButton />
    </div>
  );
}

function ExpandSidebarButton() {
  return (
    <Button
      appearance="outlined"
      className="bg-surface absolute bottom-2 right-2 z-10 rounded-md border px-2 py-1 text-xs opacity-0 shadow transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
      icon={<RiSidebarFoldLine />}
      iconSide="left"
      kind="secondary"
      label="Expand"
      onClick={(e) => {
        e.preventDefault();
        alert('test');
      }}
      size="small"
    />
  );
}
