import { interTight, robotoMono } from '@/app/fonts';
import cn from '@/utils/cn';

// This is separated from RootLayout so that we can use it in Storybook. For
// whatever reason, <PageViewTracker /> makes Storybook error.
export function BaseWrapper({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={cn('h-full bg-white accent-indigo-500', interTight.variable, robotoMono.variable)}
    >
      <body className="h-full overflow-hidden">{children}</body>
    </html>
  );
}
