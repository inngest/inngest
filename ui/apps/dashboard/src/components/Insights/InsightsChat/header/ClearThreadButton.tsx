import { Button } from '@inngest/components/Button';
import { RiEraserLine } from '@remixicon/react';

type ClearThreadButtonProps = {
  onClick: () => void;
  className?: string;
  style?: React.CSSProperties;
};

export default function ClearThreadButton({ onClick, className, style }: ClearThreadButtonProps) {
  return (
    <Button
      kind="secondary"
      appearance="outlined"
      size="small"
      icon={<RiEraserLine />}
      className={className}
      onClick={onClick}
      style={style}
    />
  );
}
