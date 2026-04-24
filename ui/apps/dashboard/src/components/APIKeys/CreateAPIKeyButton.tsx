import { Button } from '@inngest/components/Button';
import { RiAddLine } from '@remixicon/react';

type Props = {
  appearance?: 'solid' | 'outlined';
  label?: string;
  onClick: () => void;
};

// Modal state is owned by the parent route so it survives when the list
// transitions from empty -> populated (which unmounts the EmptyState).
export function CreateAPIKeyButton({
  appearance = 'solid',
  label = 'Create API key',
  onClick,
}: Props) {
  return (
    <Button
      kind="primary"
      appearance={appearance}
      icon={<RiAddLine />}
      iconSide="left"
      label={label}
      onClick={onClick}
    />
  );
}
