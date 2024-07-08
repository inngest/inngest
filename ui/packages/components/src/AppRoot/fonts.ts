import { Inter_Tight, Roboto_Mono } from 'next/font/google';

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
