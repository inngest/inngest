import { useState } from 'react';
import { Button } from '@inngest/components/Button';

type Props = {
  disabled?: boolean;
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
      onClick={onClick}
      disabled={props.disabled}
      loading={isLoading}
      label="Rerun"
      size="medium"
    />
  );
}
