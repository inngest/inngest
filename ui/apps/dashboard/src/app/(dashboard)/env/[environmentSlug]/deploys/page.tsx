import DeploysEmptyState from './DeploysEmptyState';

type DeploysProps = {
  params: {
    environmentSlug: string;
  };
};

export const runtime = 'nodejs';

export default async function Deploys({ params }: DeploysProps) {
  return <DeploysEmptyState environmentSlug={params.environmentSlug} />;
}
