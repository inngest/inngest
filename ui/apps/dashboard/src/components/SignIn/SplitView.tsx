import { cn } from '@inngest/components/utils/classNames';

export default function SplitView({
  children,
  panel,
  panelBackground = 'gradient',
}: {
  children: React.ReactNode;
  /**
   * Decorative content for the right column. Hidden below `sm`, so nothing
   * load-bearing belongs here -- put it in `children` instead.
   */
  panel?: React.ReactNode;
  /**
   * `gradient` is the long-standing treatment and stays the default.
   * `neutral` is a flat panel that sits behind the sign-up page's product
   * screenshot and logo wall, where the gradient competed with them. Opt in
   * rather than switching the default, so nothing else changes look.
   */
  panelBackground?: 'gradient' | 'neutral';
}) {
  return (
    // `dvh` rather than `vh`: on mobile Safari `100vh` resolves to the large
    // viewport, which pushes bottom-pinned content behind the browser chrome.
    <div className="flex h-dvh w-full">
      <div className="bg-canvasBase flex h-full w-full flex-col items-center overflow-y-auto py-4 sm:w-2/3 sm:p-6 md:w-1/2">
        {children}
      </div>
      <div
        className={cn(
          'hidden h-full w-1/3 sm:block md:w-1/2',
          panelBackground === 'neutral' ? 'bg-canvasMuted' : 'mesh-gradient',
        )}
      >
        {panel}
      </div>
    </div>
  );
}
