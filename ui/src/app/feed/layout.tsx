'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

import ActionBar from '@/components/ActionBar';
import Button from '@/components/Button';
import classNames from '@/utils/classnames';

type LayoutProps = {
  children: React.ReactNode;
};

export default function Feed({ children }: LayoutProps) {
  const pathname = usePathname();

  return (
    <div className="flex flex-col h-full">
      <ActionBar
        tabs={
          <>
            <Link
              href="/feed/events"
              className={classNames(
                pathname.startsWith('/feed/events')
                  ? `border-indigo-400 text-white`
                  : `border-transparent text-slate-400`,
                `text-xs px-5 py-2.5 border-b block transition-all duration-150`
              )}
            >
              Event Stream
            </Link>
            <Link
              href="/feed/functions"
              className={classNames(
                pathname.startsWith('/feed/functions')
                  ? `border-indigo-400 text-white`
                  : `border-transparent text-slate-400`,
                `text-xs px-5 py-2.5 border-b block transition-all duration-150`
              )}
            >
              Function Log
            </Link>
          </>
        }
        actions={<Button label="Send event" />}
      />

      {children}
    </div>
  );
}
