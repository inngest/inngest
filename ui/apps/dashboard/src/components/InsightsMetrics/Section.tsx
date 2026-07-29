import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiArrowRightUpLine, RiMoreFill } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator } from '@/utils/urls';

export function SectionGroupHeading({ children }: { children: React.ReactNode }) {
  return <h2 className="text-basis mb-3 mt-6 text-lg font-normal">{children}</h2>;
}

export function Section({
  title,
  className,
  children,
  query,
  queryName,
  plain = false,
}: {
  title?: string;
  className?: string;
  children: React.ReactNode;
  // The exact Insights-dialect SQL that produced this card's data (the
  // insightsMetric result's `query` field) — when present, an "Open in
  // Insights" link opens that same query for the user to inspect/modify.
  query?: string;
  queryName?: string;
  // Skip the bordered/background card chrome — just the header row and
  // children, unboxed.
  plain?: boolean;
}) {
  const env = useEnvironment();
  const navigate = useNavigate();

  return (
    <section
      // No margin here — every Section sits inside a `gap-4` grid/flex
      // container (see the surrounding grids below), which already spaces
      // rows/columns evenly at 16px; an own-margin here would double the
      // vertical gap without affecting the horizontal one.
      className={`${plain ? '' : 'border-subtle bg-canvasBase shadow-xs rounded-md border p-4'} ${className ?? ''}`}
    >
      <div className="mb-3 flex items-center justify-between gap-2">
        {title && <h2 className="text-basis text-sm font-medium">{title}</h2>}
        {query && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild className="ml-auto">
              <Button size="small" kind="secondary" appearance="ghost" icon={<RiMoreFill />} />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onSelect={() =>
                  navigate({
                    to: pathCreator.insights({ envSlug: env.slug }),
                    search: { sql: query, name: queryName ?? title },
                  })
                }
              >
                <RiArrowRightUpLine className="h-4 w-4" />
                Open in Insights
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      {children}
    </section>
  );
}
