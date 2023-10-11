import Font from 'next/font/local';

import cn from '@/utils/cn';

const inter = Font({
  src: '../fonts/InterTight-VariableFont.woff2',
  variable: '--font-inter',
});

const robotoMono = Font({
  src: '../fonts/RobotoMono-VariableFont.woff2',
  variable: '--font-roboto-mono',
});

// This is separated from RootLayout so that we can use it in Storybook. For
// whatever reason, <PageViewTracker /> makes Storybook error.
export function BaseWrapper({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={cn('h-full bg-white accent-indigo-500', inter.variable, robotoMono.variable)}
    >
      <body className="h-full overflow-hidden">{children}</body>
    </html>
  );
}
