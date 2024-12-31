import type { Route } from 'next';
import { NewLink } from '@inngest/components/Link';

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
      <NewLink href={vercelDeploymentURL as Route} target="_blank" size="medium">
        <span className="truncate">{vercelDeploymentID}</span>
      </NewLink>
    );
  } else {
    deploymentValue = '-';
  }

  let projectValue;
  if (vercelProjectID && vercelProjectURL) {
    projectValue = (
      <NewLink href={vercelProjectURL as Route} target="_blank" size="medium">
        <span className="truncate">{vercelProjectID}</span>
      </NewLink>
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
