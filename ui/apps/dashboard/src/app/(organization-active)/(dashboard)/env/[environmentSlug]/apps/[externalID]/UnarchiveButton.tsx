import { NewButton } from '@inngest/components/Button';
import { RiArchive2Line } from '@remixicon/react';

type Props = {
  showArchive: () => void;
};

export function UnarchiveButton({ showArchive }: Props) {
  return (
    <>
      <NewButton
        onClick={showArchive}
        kind="danger"
        label="Unarchive app"
        icon={<RiArchive2Line />}
        iconSide="left"
      />
    </>
  );
}
