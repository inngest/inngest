import type { Route } from 'next';
import NextLink from 'next/link';
import { Link, type LinkProps } from '@inngest/components/Link';
import { RiCheckboxCircleFill, type RemixiconComponentType } from '@remixicon/react';

import { cn } from '../utils/classNames';

export type MenuStepContent = {
  title: string;
  description: string;
  icon: RemixiconComponentType;
};

export default function StepsMenu({
  children,
  title,
  links,
}: React.PropsWithChildren<{ title: string; links?: React.ReactNode }>) {
  return (
    <div className="mr-12 flex flex-col">
      <nav className="mb-12">
        <h3 className="text-subtle text-xs font-medium uppercase">{title}</h3>
        <ul className="my-2">{children}</ul>
      </nav>
      {links}
    </div>
  );
}

function StepMenuItem({
  stepContent,
  isCompleted,
  isActive,
  isDisabled,
  url,
}: {
  stepContent: MenuStepContent;
  isCompleted: boolean;
  isActive: boolean;
  isDisabled?: boolean;
  url: Route;
}) {
  const { title, description, icon: Icon } = stepContent;

  const content = (
    <li
      className={cn(
        'flex items-center gap-4 rounded-md p-1.5',
        isDisabled
          ? 'bg-canvasBase cursor-not-allowed opacity-50'
          : 'bg-canvasBase hover:bg-canvasSubtle group cursor-pointer'
      )}
    >
      <div
        className={cn(
          'box-border flex h-[38px] w-[38px] items-center justify-center rounded-md border',
          isDisabled
            ? 'border-muted'
            : isActive
            ? isCompleted
              ? 'border-primary-moderate bg-primary-3xSubtle group-hover:bg-primary-moderate'
              : 'border-contrast group-hover:bg-contrast'
            : isCompleted
            ? 'bg-primary-3xSubtle group-hover:bg-primary-moderate border-none'
            : 'border-muted group-hover:bg-contrast group-hover:border-none'
        )}
      >
        {isCompleted ? (
          <RiCheckboxCircleFill
            className={cn(
              'text-primary-moderate h-5 w-5',
              !isDisabled && 'group-hover:text-alwaysWhite'
            )}
          />
        ) : (
          <Icon className={cn('h-4 w-4', !isDisabled && 'group-hover:text-onContrast')} />
        )}
      </div>
      <div>
        <h4 className="text-sm font-medium">{title}</h4>
        <p className="text-subtle text-sm">{description}</p>
      </div>
    </li>
  );

  return isDisabled ? content : <NextLink href={url}>{content}</NextLink>;
}

function StepLink({ children, href, ...props }: React.PropsWithChildren<LinkProps>) {
  return (
    <Link
      className="text-subtle hover:decoration-subtle mx-1.5 my-1"
      href={href}
      size="small"
      {...props}
    >
      {children}
    </Link>
  );
}

StepsMenu.MenuItem = StepMenuItem;
StepsMenu.Link = StepLink;
