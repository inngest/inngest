import { RiRefreshLine } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';
import { toast } from 'sonner';

import { Button } from '../Button/NewButton';

export const RefreshButton = () => {
  const navigate = useNavigate();

  return (
    <Button
      kind="primary"
      appearance="outlined"
      label="Refresh page"
      icon={<RiRefreshLine />}
      iconSide="left"
      onClick={() => {
        navigate({ to: '.' });
        setTimeout(() => toast.success('Page successfully refreshed!'), 500);
      }}
    />
  );
};
