import { Button } from '@inngest/components/Button';

import { Alert } from '@/components/Alert';

export default function ApprovalDialog({
  title,
  description,
  secondaryInfo,
  graphic,
  onCancel,
  onApprove,
  isLoading = false,
  error = '',
}: {
  title: string;
  description: React.ReactNode;
  secondaryInfo: React.ReactNode;
  graphic: React.ReactNode;
  onCancel: () => void;
  onApprove: () => void;
  isLoading: boolean;
  error?: string | React.ReactNode;
}) {
  return (
    <main className="m-auto max-w-2xl pb-24 text-center font-medium">
      <h2 className="my-6 text-xl font-bold">{title}</h2>
      <div className="my-12 flex flex-row place-content-center items-center justify-items-center gap-6">
        {graphic}
      </div>
      <div className="mx-auto max-w-xl">{description}</div>
      <div className="my-12 flex justify-center gap-6">
        <Button
          btnAction={onCancel}
          appearance="outlined"
          size="large"
          disabled={isLoading}
          label="Cancel"
        />
        <Button
          btnAction={onApprove}
          kind="primary"
          size="large"
          disabled={isLoading}
          label="Approve"
        />
      </div>
      {error && <Alert severity="error">{error}</Alert>}
      <p className="mt-12 text-sm text-slate-500">{secondaryInfo}</p>
    </main>
  );
}
