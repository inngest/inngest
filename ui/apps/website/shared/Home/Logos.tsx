import Image from 'next/image';
import Link from 'next/link';
import clsx from 'clsx';

import Container from '../layout/Container';

export default function Logos({
  heading,
  logos,
  footer,
  variant = 'dark',
  className,
}: {
  heading?: string | React.ReactNode;
  logos: {
    src: string;
    name: string;
    href?: string;
    featured?: boolean;
    scale?: number;
  }[];
  footer?: React.ReactNode;
  className?: string;
  variant?: 'dark' | 'light';
}) {
  const hasLinks = !!logos.find((l) => !!l.href);
  const nonFeaturedCount = logos.filter((l) => !l.featured).length;
  return (
    <Container
      className={clsx(
        'mx-auto max-w-4xl', // my-20 lg:my-36 mb-20 lg:mb-40 xl:mb-60
        className
      )}
    >
      {!!heading && (
        <h2
          className={clsx(
            'text-center text-lg tracking-tight',
            variant === 'dark' && 'text-slate-400 drop-shadow',
            variant === 'light' && 'text-slate-700'
          )}
        >
          {heading}
        </h2>
      )}
      <div
        className={clsx(
          'm-auto mt-16 grid max-w-[1200px] grid-cols-2 items-center justify-center',
          nonFeaturedCount === 4 && 'sm:px-8 md:px-20 lg:grid-cols-4',
          nonFeaturedCount === 5 && 'sm:px-6 lg:grid-cols-5',
          hasLinks ? 'gap-x-4 gap-y-8' : 'gap-x-16 gap-y-16',
          footer && 'mb-16'
        )}
      >
        {logos.map(({ src, name, href, featured, scale = 1 }, idx) => {
          if (href) {
            return (
              <Link
                href={href}
                className={clsx(
                  'group m-auto flex h-16 w-40 max-w-[90%] items-center justify-center rounded-lg border px-6 py-6 transition-all',
                  variant === 'dark' && 'border-slate-700 hover:border-slate-600',
                  variant === 'light' && 'border-slate-200 hover:border-slate-300',
                  featured && 'col-span-2',
                  !featured &&
                    nonFeaturedCount % 2 == 1 &&
                    idx === logos.length - 1 &&
                    'col-span-2 lg:col-span-1' // center the last item if there is an odd number
                )}
              >
                <Image
                  key={idx}
                  src={src}
                  alt={name}
                  width={120}
                  height={30}
                  className="pointer-events-none max-h-[40px] text-white opacity-80 transition-all group-hover:opacity-100"
                />
              </Link>
            );
          }
          return (
            <Image
              key={idx}
              src={src}
              alt={name}
              title={name}
              width={(featured ? 240 : 120) * scale}
              height={(featured ? 120 : 30) * scale}
              className={clsx(
                'width-auto m-auto text-white grayscale transition-all hover:grayscale-0',
                `max-h-[${36 * scale}px]`,
                featured && 'col-span-2 max-h-[60px]'
              )}
            />
          );
        })}
      </div>

      {footer}
    </Container>
  );
}
