import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiLoopLeftLine } from '@remixicon/react';
import { toast } from 'sonner';

type Props = {
  onClick: () => Promise<unknown>;
};

/**
 * @deprecated Delete this when the old run details page is removed
 */
export function RerunButton(props: Props) {
  const [isLoading, setIsLoading] = useState(false);

  async function onClick() {
    setIsLoading(true);
    try {
      await props.onClick();
      toast.success('Queued rerun');
    } catch {
      toast.error('Failed to queue rerun');
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <Button
      onClick={onClick}
      disabled={isLoading}
      icon={<RiLoopLeftLine />}
      iconSide="left"
      label="Rerun"
      appearance="outlined"
      size="medium"
    />
  );
}
