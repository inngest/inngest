import { Link } from '@inngest/components/Link';
import { RiExternalLinkLine } from '@remixicon/react';

export function CompactDeploymentGuide({
  onLinkClick,
}: {
  onLinkClick: (action: 'cloud_clicked' | 'docs_clicked') => void;
}) {
  return (
    <div className="pt-3">
      <p className="text-muted mb-5 text-sm">
        Inngest Cloud coordinates and monitors your workflows in production. You
        can deploy your app to any hosting provider.
      </p>
      <ol className="text-muted list-inside list-decimal space-y-2 text-sm">
        <li>
          Sign in to{' '}
          <Link
            className="inline"
            href="https://app.inngest.com"
            onClick={() => onLinkClick('cloud_clicked')}
          >
            Cloud
          </Link>{' '}
          and choose your environment.
        </li>
        <li>Add your signing and event keys, then deploy.</li>
        <li>Sync your deployed endpoint in Cloud.</li>
      </ol>
      <div className="border-subtle mt-5 border-t pt-4">
        <Link
          href="https://www.inngest.com/docs/durable-execution/deploying-functions/platforms"
          onClick={() => onLinkClick('docs_clicked')}
          iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
        >
          Full deployment guide
        </Link>
      </div>
    </div>
  );
}
