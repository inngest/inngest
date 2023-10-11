import cn from '@/utils/cn';

type BlockProps = {
  title?: string;
  className?: string;
  children: React.ReactNode;
};

export default function Block({ title, children, className = '' }: BlockProps) {
  return (
    <div className={className}>
      {title ? <h2 className="mb-2 text-base font-medium text-slate-800">{title}</h2> : ''}
      {children}
    </div>
  );
}
