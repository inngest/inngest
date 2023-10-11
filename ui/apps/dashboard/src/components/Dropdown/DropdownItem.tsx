'use client';

import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';
import { Menu } from '@headlessui/react';

import cn from '@/utils/cn';

export type DropdownItemComponentProps = {
  className: string;
  children: React.ReactNode;
};

type DropdownItemProps<PassedHref extends string> = {
  children: React.ReactNode;
  context?: 'dark' | 'light';
  href?: Route<PassedHref> | UrlObject;
  target?: string;
  rel?: string;
  Component?: React.ComponentType<DropdownItemComponentProps>;
};

const contextStyles = {
  light: 'text-slate-500 hover:text-slate-800 font-medium',
  dark: 'text-slate-400 hover:text-white',
};

export default function DropdownItem<PassedHref extends string>({
  children,
  context = 'light',
  href,
  target,
  rel,
  Component,
}: DropdownItemProps<PassedHref>) {
  const className = cn(
    'block px-2 py-2 flex items-center gap-2 text-sm text-left font-medium',
    contextStyles[context]
  );
  if (Component) {
    return (
      <Menu.Item>
        <Component className={className}>{children}</Component>
      </Menu.Item>
    );
  }

  if (!href) {
    throw new Error('href is required for non-component DropdownItems');
  }

  return (
    <Menu.Item>
      <Link href={href} target={target} rel={rel} className={className}>
        {children}
      </Link>
    </Menu.Item>
  );
}
