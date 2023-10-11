import { type Route } from 'next';
import Link from 'next/link';

import { Pill } from '@/components/Pill/Pill';
import Status, { type StatusTypeKind } from '@/components/Status/Status';

interface Props {
  name: string;
  href: string;
  status: 'active' | 'removed';
}

export function FunctionListItem({ name, href, status }: Props) {
  let statusKind: StatusTypeKind = 'success';
  let statusText = 'Active';

  if (status === 'removed') {
    statusKind = 'error';
    statusText = 'Removed';
  }

  return (
    <li key={name}>
      <Link
        href={href as Route}
        className="flex items-center justify-start gap-4 truncate px-4 py-2 text-sm text-white hover:bg-slate-800"
      >
        <Pill variant="dark">
          <Status kind={statusKind} className="text-white">
            {statusText}
          </Status>
        </Pill>
        <span className="truncate">{name}</span>
      </Link>
    </li>
  );
}
