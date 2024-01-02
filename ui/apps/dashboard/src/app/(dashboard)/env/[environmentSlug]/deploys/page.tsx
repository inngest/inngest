import DeploysEmptyState from './DeploysEmptyState';

export const runtime = 'nodejs';

export default async function Deploys() {
  return <DeploysEmptyState />;
}
