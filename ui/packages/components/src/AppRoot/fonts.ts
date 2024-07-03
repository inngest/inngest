import { Inter_Tight, Roboto_Mono } from 'next/font/google';
import localFont from 'next/font/local';

export const circular = localFont({
  variable: '--font-circular',
  src: [
    {
      path: './fonts/CircularXXWeb-Regular.woff2',
      weight: '100 400',
    },

    {
      path: './fonts/CircularXXWeb-Medium.woff2',
      weight: '500 900',
    },
  ],
});

export const circularMono = localFont({
  variable: '--font-circular-mono',
  src: [
    {
      path: './fonts/CircularXXMonoWeb-Regular.woff2',
      weight: '100 900',
    },
  ],
});

/**
 * TODO: these are deprecated and can be remove once we have
 * transitioned everything to Circular
 */
export const interTight = Inter_Tight({
  subsets: ['latin'],
  variable: '--font-inter-tight',
});

export const robotoMono = Roboto_Mono({
  subsets: ['latin'],
  variable: '--font-roboto-mono',
});
