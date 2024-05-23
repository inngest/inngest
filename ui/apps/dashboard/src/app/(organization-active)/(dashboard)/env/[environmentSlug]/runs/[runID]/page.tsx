import { RunDetails } from '@/components/RunDetails/RunDetails';

type Props = {
  params: {
    runID: string;
  };
};

export default function Page({ params }: Props) {
  const runID = decodeURIComponent(params.runID);
  return <RunDetails runID={runID} />;
}
