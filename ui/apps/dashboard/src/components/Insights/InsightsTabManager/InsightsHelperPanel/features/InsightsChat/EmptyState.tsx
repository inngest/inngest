import { Link } from '@inngest/components/Link';
import { RiChatPollLine } from '@remixicon/react';

export function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center py-12">
      <div className="flex max-w-[410px] flex-col items-center gap-2">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiChatPollLine className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">
            What can I help you query?
          </h3>
          <p className="text-muted text-sm">
            Share the event insights you need and our AI assistant will generate
            the SQL query.{' '}
            <span>
              {' '}
              <Link
                className="text-link inline-flex text-sm"
                href="https://www.inngest.com/docs/platform/monitor/insights"
                target="_blank"
                rel="noopener"
              >
                Learn more
              </Link>
            </span>
          </p>
        </div>
      </div>
    </div>
  );
}
