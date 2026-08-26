export default function SplitView({
  children,
  panel,
}: {
  children: React.ReactNode;
  /**
   * Decorative content for the right column. Hidden below `sm`, so nothing
   * load-bearing belongs here -- put it in `children` instead.
   */
  panel?: React.ReactNode;
}) {
  return (
    // `dvh` rather than `vh`: on mobile Safari `100vh` resolves to the large
    // viewport, which pushes bottom-pinned content behind the browser chrome.
    <div className="flex h-dvh w-full">
      <div className="bg-canvasBase flex h-full w-full flex-col items-center overflow-y-auto py-4 sm:w-2/3 sm:p-6 md:w-1/2">
        {children}
      </div>
      <div className="mesh-gradient hidden h-full w-1/3 sm:block md:w-1/2">
        {panel}
      </div>
    </div>
  );
}
