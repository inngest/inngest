import { type ReactNode } from "react";

export default function SplitView({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-screen w-full items-center justify-center bg-canvasBase">
      {children}
    </div>
  );
}
