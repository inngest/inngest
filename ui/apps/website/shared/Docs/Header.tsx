import { forwardRef } from 'react';
import Link from 'next/link';
import clsx from 'clsx';
import { motion, useScroll, useTransform } from 'framer-motion';

import { Button } from '../Button';
import Logo from '../Icons/Logo';
import {
  MobileNavigation,
  useIsInsideMobileNavigation,
  useMobileNavigationStore,
} from './MobileNavigation';
import { ModeToggle } from './ModeToggle';
import { headerLinks } from './Navigation';
import { MobileSearch, Search } from './Search';
import SocialBadges from './SocialBadges';

function TopLevelNavItem({ href, children }) {
  return (
    <li>
      <a
        href={href}
        className="text-sm leading-5 text-slate-600 transition hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
      >
        {children}
      </a>
    </li>
  );
}

export const Header = forwardRef<HTMLDivElement>(function Header(
  { className }: { className?: string },
  ref
) {
  let { isOpen: mobileNavIsOpen } = useMobileNavigationStore();
  let isInsideMobileNavigation = useIsInsideMobileNavigation();

  let { scrollY } = useScroll();
  let bgOpacityLight = useTransform(scrollY, [0, 72], [0.5, 0.9]);
  let bgOpacityDark = useTransform(scrollY, [0, 72], [0.2, 0.8]);

  return (
    <motion.div
      ref={ref}
      className={clsx(
        className,
        // NOTE - if we remove the AI button we may have to add "lg:justify-end"
        'fixed inset-x-0 top-0 z-50 flex h-14 items-center justify-between gap-12 px-4 transition sm:px-6 lg:z-30 lg:px-8',
        !isInsideMobileNavigation && 'backdrop-blur-sm lg:left-72 xl:left-80 dark:backdrop-blur',
        isInsideMobileNavigation
          ? 'bg-white dark:bg-slate-900'
          : 'bg-white/[var(--bg-opacity-light)] dark:bg-slate-950/[var(--bg-opacity-dark)]'
      )}
      style={
        {
          '--bg-opacity-light': bgOpacityLight,
          '--bg-opacity-dark': bgOpacityDark,
        } as any
      }
    >
      <div
        className={clsx(
          'absolute inset-x-0 top-full h-px transition',
          (isInsideMobileNavigation || !mobileNavIsOpen) && 'bg-slate-900/7.5 dark:bg-white/7.5'
        )}
      />
      <Search />
      <div className="flex items-center gap-5 lg:hidden">
        <MobileNavigation />
        <a href="/" className="group/logo flex items-center gap-1.5 pt-1">
          <Logo className="w-20 text-indigo-500 dark:text-white" />
          <span className="transition-color text-base font-semibold text-slate-700 group-hover/logo:text-white dark:text-indigo-400">
            Docs
          </span>
        </a>
      </div>
      <div className="flex items-center gap-5">
        <nav className="mr-4 hidden lg:block">
          <ul role="list" className="flex items-center gap-8">
            {headerLinks.map((link) => (
              <TopLevelNavItem key={link.title} href={link.href}>
                {link.title}
              </TopLevelNavItem>
            ))}
          </ul>
        </nav>
        <div className="hidden lg:block">
          <SocialBadges />
        </div>
        <div className="hidden md:h-5 md:w-px md:bg-slate-900/10 lg:block md:dark:bg-white/15" />
        <div className="flex gap-4">
          <MobileSearch />
          <ModeToggle />
        </div>
        <div className="hidden items-center gap-3 sm:flex">
          <Button href={process.env.NEXT_PUBLIC_SIGNIN_URL} size="sm" variant="secondary">
            Sign In
          </Button>
          <Button
            href={`${process.env.NEXT_PUBLIC_SIGNUP_URL}?ref=docs-header`}
            size="sm"
            arrow="right"
          >
            Sign Up
          </Button>
        </div>
      </div>
    </motion.div>
  );
});
