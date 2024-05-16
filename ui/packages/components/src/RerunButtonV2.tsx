import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { RiLoopLeftLine } from '@remixicon/react';

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
    } catch {
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
