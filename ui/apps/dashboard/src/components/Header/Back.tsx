import { NewButton } from '@inngest/components/Button';
import { RiArrowLeftLine } from '@remixicon/react';

export const Back = ({ className }: { className?: string }) => {
  return (
    <NewButton
      kind="secondary"
      appearance="outlined"
      size="small"
      icon={<RiArrowLeftLine />}
      className={className}
      href="/"
      scroll={false}
    />
  );
};
