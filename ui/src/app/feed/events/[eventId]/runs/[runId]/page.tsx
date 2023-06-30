import { FunctionRunSection } from '@/components/Function/RunSection';

export default function ShowRun({ params }: { params: { runId: string } }) {
  return <FunctionRunSection runId={params.runId} />;
}
