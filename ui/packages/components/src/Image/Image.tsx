import Image, { type ImageProps } from 'next/image';
import { cn } from '@inngest/components/utils/classNames';

interface ThemeImageProps extends Omit<ImageProps, 'src'> {
  lightSrc: string | any; // Support both string URLs and imported images
  darkSrc: string | any;
  alt: string;
  className?: string;
}

export function ThemeImage({ lightSrc, darkSrc, alt, className = '', ...props }: ThemeImageProps) {
  return (
    <>
      <Image
        src={lightSrc}
        alt={alt}
        className={cn('hidden md:block dark:hidden', className)}
        {...props}
      />
      <Image src={darkSrc} alt={alt} className={cn('hidden dark:md:block', className)} {...props} />
    </>
  );
}
