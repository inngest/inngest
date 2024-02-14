import clsx from 'clsx';

export default function Heading({
  title,
  lede,
  variant = 'dark',
  className,
}: {
  title: React.ReactNode;
  lede?: React.ReactNode;
  variant?: 'dark' | 'light';
  className?: string;
}) {
  return (
    <div className={`${className}`}>
      <h2
        className={clsx(
          'text-2xl font-semibold leading-snug tracking-tight md:text-5xl ',
          variant === 'dark' &&
            'bg-gradient-to-br from-white to-slate-300 bg-clip-text text-transparent',
          variant === 'light' && 'text-slate-800'
        )}
      >
        {title}
      </h2>
      {!!lede && (
        <p
          className={clsx(
            'my-4 text-sm font-medium leading-loose sm:text-base md:text-lg',
            variant === 'dark' && 'text-indigo-100/90',
            variant === 'light' && 'font-medium text-slate-500'
          )}
        >
          {lede}
        </p>
      )}
    </div>
  );
}
