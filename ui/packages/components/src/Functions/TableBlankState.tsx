import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';

import { InlineCode } from '../Code';
import { TableBlankState } from '../Table/TableBlankState';

type TableBlankStateProps = {
  actions: React.ReactNode;
  title?: string;
};

export default function BlankState({ actions, title }: TableBlankStateProps) {
  return (
    <TableBlankState
      icon={<FunctionsIcon />}
      actions={actions}
      title={title || 'No functions found'}
      description={
        <>
          To create functions, you will use <InlineCode>inngest.createFunction()</InlineCode>.
        </>
      }
    />
  );
}
