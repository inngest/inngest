import type { Route } from 'next';
import Link from 'next/link';
import { NewLink, type NewLinkProps } from '@inngest/components/Link';
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
        <h3 className="text-muted text-xs font-medium uppercase">{title}</h3>
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
  url,
}: {
  stepContent: MenuStepContent;
  isCompleted: boolean;
  isActive: boolean;
  url: Route;
}) {
  const { title, description, icon: Icon } = stepContent;
  return (
    <Link href={url}>
      <li className="bg-canvasBase hover:bg-canvasSubtle group flex items-center gap-4 rounded-md p-1.5">
        <div
          className={cn(
            'group-hover:bg-contrast box-border flex h-[38px] w-[38px] items-center justify-center rounded-md border group-hover:border-none',
            isActive
              ? isCompleted
                ? 'border-primary-moderate bg-primary-3xSubtle group-hover:bg-primary-moderate'
                : 'border-contrast'
              : isCompleted
              ? 'bg-primary-3xSubtle group-hover:bg-primary-moderate border-none'
              : 'border-muted'
          )}
        >
          {isCompleted ? (
            <RiCheckboxCircleFill className="text-primary-moderate group-hover:text-alwaysWhite" />
          ) : (
            <Icon className="group-hover:text-onContrast h-5 w-5" />
          )}
        </div>
        <div>
          <h4 className="text-sm font-medium">{title}</h4>
          <p className="text-muted text-sm">{description}</p>
        </div>
      </li>
    </Link>
  );
}

function StepLink({ children, href, ...props }: React.PropsWithChildren<NewLinkProps>) {
  return (
    <NewLink
      className="text-muted hover:decoration-subtle mx-1.5 my-1"
      href={href}
      size="small"
      {...props}
    >
      {children}
    </NewLink>
  );
}

StepsMenu.MenuItem = StepMenuItem;
StepsMenu.Link = StepLink;
