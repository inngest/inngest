import { useSystemStatus } from '@/app/(organization-active)/support/statusPage';
import cn from '@/utils/cn';

type SystemStatusIconProps = {
  className?: string;
};

export default function SystemStatusIcon({ className }: SystemStatusIconProps) {
  const status = useSystemStatus();

  return (
    <span
      className={cn('mx-1 inline-flex h-2 w-2 rounded-full', className)}
      style={{ backgroundColor: status.indicatorColor }}
      title={`Status updated at ${status.updated_at}`}
    />
  );
}
