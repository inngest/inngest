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
      <a
        className="text-indigo-600 hover:underline"
        href={vercelDeploymentURL}
        rel="noopener noreferrer"
        target="_blank"
      >
        {vercelDeploymentID}
      </a>
    );
  } else {
    deploymentValue = '-';
  }

  let projectValue;
  if (vercelProjectID && vercelProjectURL) {
    projectValue = (
      <a
        className="text-indigo-600 hover:underline"
        href={vercelProjectURL}
        rel="noopener noreferrer"
        target="_blank"
      >
        {vercelProjectID}
      </a>
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
      <Description className="truncate" detail={projectValue} term="Vercel Project" />
      <Description className="truncate" detail={deploymentValue} term="Vercel Deployment" />
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
      <dt className="text-xs text-slate-600">{term}</dt>
      <dd>{detail}</dd>
    </div>
  );
}
