import { Link } from '@inngest/components/Link';
import { RiExternalLinkLine } from '@remixicon/react';

export function CompactDeploymentGuide({
  descriptionID,
  onLinkClick,
}: {
  descriptionID?: string;
  onLinkClick: (action: 'cloud_clicked' | 'docs_clicked') => void;
}) {
  return (
    <>
      <p id={descriptionID} className="text-muted mb-2 text-xs leading-4">
        Inngest Cloud coordinates and monitors your workflows in production. You
        can deploy your app to any hosting provider.
      </p>
      <ol className="text-muted mt-auto list-outside list-decimal space-y-1 pl-5 text-xs leading-4">
        <li>
          Sign in to{' '}
          <Link
            className="inline text-xs leading-4"
            href="https://app.inngest.com"
            onClick={() => onLinkClick('cloud_clicked')}
          >
            Cloud
          </Link>{' '}
          and choose your environment.
        </li>
        <li>Add your signing and event keys, then deploy.</li>
        <li>
          Sync your deployed HTTP endpoint in Cloud. If you use Connect, start
          your workers and confirm they appear in Cloud.
        </li>
      </ol>
      <div className="mt-auto pt-2">
        <Link
          className="text-xs leading-4"
          href="https://www.inngest.com/docs/durable-execution/deploying-functions/platforms"
          onClick={() => onLinkClick('docs_clicked')}
          iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
        >
          Full deployment guide
        </Link>
      </div>
    </>
  );
}
