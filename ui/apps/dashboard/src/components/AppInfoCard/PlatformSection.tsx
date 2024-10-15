import type { Route } from 'next';
import { Link } from '@inngest/components/Link';

import { PlatformInfo } from '@/components/PlatformInfo';
import { CardItem } from './CardItem';

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
        <span className="truncate">{vercelDeploymentID}</span>
      </Link>
    );
  } else {
    deploymentValue = '-';
  }

  let projectValue;
  if (vercelProjectID && vercelProjectURL) {
    projectValue = (
      <Link href={vercelProjectURL as Route} internalNavigation={false}>
        <span className="truncate">{vercelProjectID}</span>
      </Link>
    );
  } else {
    projectValue = '-';
  }

  return (
    <>
      <CardItem detail={<PlatformInfo platform={platform} />} term="Platform" />
      <CardItem detail={projectValue} term="Vercel Project" />
      <CardItem detail={deploymentValue} term="Vercel Deployment" />
    </>
  );
}
