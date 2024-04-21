import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiLoopLeftLine } from '@remixicon/react';
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
    <Button
      btnAction={onClick}
      disabled={isLoading}
      icon={<RiLoopLeftLine className={cn(' text-sky-500', isLoading && 'animate-spin')} />}
      label="Rerun"
      size="small"
    />
  );
}
