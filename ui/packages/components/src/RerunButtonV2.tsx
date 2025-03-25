import { Button } from '@inngest/components/Button';

type Props = {
  disabled: boolean;
  loading: boolean;
  rerun: () => void;
};

export function RerunButton({ disabled, loading, rerun }: Props) {
  return (
    <Button
      onClick={rerun}
      disabled={disabled || loading}
      loading={loading}
      label="Rerun"
      size="medium"
    />
  );
}
