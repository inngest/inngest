'use client';

import { type Route } from 'next';

import { useSystemStatus } from '@/app/support/statusPage';
import DropdownItem from '../Dropdown/DropdownItem';

export default function StatusPageItem() {
  const status = useSystemStatus();

  return (
    <DropdownItem
      context="dark"
      href={status.url as Route}
      target="_blank"
      rel="noopener noreferrer"
    >
      <span
        className={`mx-1 inline-flex h-2 w-2 rounded-full`}
        style={{ backgroundColor: status.indicatorColor }}
        title={`Status updated at ${status.updated_at}`}
      ></span>
      Status Page
    </DropdownItem>
  );
}
