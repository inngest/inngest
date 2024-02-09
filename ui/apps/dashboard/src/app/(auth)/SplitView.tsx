export default function SplitView({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen w-screen">
      <div className="flex h-full w-full flex-col items-center justify-items-center overflow-y-scroll bg-white py-4 sm:w-2/3 sm:p-6 md:w-1/2">
        {children}
      </div>
      <div className="mesh-gradient hidden w-1/3 sm:block md:w-1/2"></div>
    </div>
  );
}
