import { DashboardRunDetails } from '@/components/RunDetails/RunDetails';

type Props = {
  params: Promise<{
    runID: string;
  }>;
};

export default async function Page(props: Props) {
  const params = await props.params;
  const runID = decodeURIComponent(params.runID);
  return <DashboardRunDetails runID={runID} />;
}
