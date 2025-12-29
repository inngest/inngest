import { RiRefreshLine } from '@remixicon/react';
import { useRouter } from '@tanstack/react-router';
import { toast } from 'sonner';

import { Button } from '../Button/NewButton';

export const RefreshButton = () => {
  const router = useRouter();

  return (
    <Button
      kind="primary"
      appearance="outlined"
      label="Refresh page"
      icon={<RiRefreshLine />}
      iconSide="left"
      onClick={() => {
        router.invalidate();
        setTimeout(() => toast.success('Page successfully refreshed!'), 500);
      }}
    />
  );
};
