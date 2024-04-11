import { useState } from 'react';
import { ArrowPathIcon } from '@heroicons/react/20/solid';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';

import { cn } from './utils/classNames';

type Props = {
  onClick: () => Promise<unknown>;
};

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
    <>
      <Button
        btnAction={onClick}
        disabled={isLoading}
        icon={<ArrowPathIcon className={cn(' text-sky-500', isLoading && 'animate-spin')} />}
        label="Rerun"
        size="small"
      />
    </>
  );
}
