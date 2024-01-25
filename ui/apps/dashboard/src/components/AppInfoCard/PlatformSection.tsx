import type { Route } from 'next';
import { Link } from '@inngest/components/Link';

import { PlatformInfo } from '@/components/PlatformInfo';

type Props = {
  sync: {
    platform: string | null;
    vercelDeploymentID: string | null;
    vercelDeploymentURL: string | null;
    vercelProjectID: string | null;
    vercelProjectURL: string | null;
  };
};

export function PlatformSection({ sync }: Props) {
  const { platform, vercelDeploymentID, vercelDeploymentURL, vercelProjectID, vercelProjectURL } =
    sync;
  if (!platform) {
    return null;
  }

  let deploymentValue;
  if (vercelDeploymentID && vercelDeploymentURL) {
    deploymentValue = (
      <Link href={vercelDeploymentURL as Route} internalNavigation={false}>
        <span className="flex-1 truncate">{vercelDeploymentID}</span>
      </Link>
    );
  } else {
    deploymentValue = '-';
  }

  let projectValue;
  if (vercelProjectID && vercelProjectURL) {
    projectValue = (
      <Link href={vercelProjectURL as Route} internalNavigation={false}>
        <span className="flex-1 truncate">{vercelProjectID}</span>
      </Link>
    );
  } else {
    projectValue = '-';
  }

  return (
    <>
      <Description
        className="truncate"
        detail={<PlatformInfo platform={platform} />}
        term="Platform"
      />
      <Description detail={projectValue} term="Vercel Project" />
      <Description detail={deploymentValue} term="Vercel Deployment" />
    </>
  );
}

function Description({
  className,
  detail,
  term,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
}) {
  return (
    <div className={className}>
      <dt className="pb-2 text-sm text-slate-400">{term}</dt>
      <dd className="text-slate-800">{detail}</dd>
    </div>
  );
}
