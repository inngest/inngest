import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { Panel } from './shared';

function Placeholder({ className }: { className: string }) {
  return (
    <Skeleton
      className={cn(
        'bg-canvasMuted/50 motion-reduce:before:animate-none',
        className,
      )}
    />
  );
}

export function VerificationSkeleton() {
  return (
    <div role="status" aria-busy="true" aria-label="Loading verification">
      <span className="sr-only">Loading verification…</span>
      <div aria-hidden="true">
        <Panel>
          <Placeholder className="h-[34px] w-[34px] rounded-md" />
          <div className="flex flex-col gap-1">
            <Placeholder className="h-4 w-16" />
            <Placeholder className="h-8 w-3/5 rounded" />
            <Placeholder className="h-5 w-full" />
          </div>
          <div className="flex flex-col gap-5">
            <div className="flex flex-col gap-[5px]">
              <Placeholder className="h-5 w-24" />
              <div className="grid grid-cols-[78px_minmax(0,1fr)] gap-2">
                <Placeholder className="h-10 rounded" />
                <Placeholder className="h-10 rounded" />
              </div>
            </div>
            <Placeholder className="h-10 rounded-md" />
          </div>
        </Panel>
      </div>
    </div>
  );
}
