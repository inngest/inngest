import { TextCell } from '@inngest/components/Table';

interface JSONAwareTextCellProps {
  children: string;
}

export function JSONAwareTextCell({ children }: JSONAwareTextCellProps) {
  return <TextCell>{children}</TextCell>;
}
