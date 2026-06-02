import { cn } from "@inngest/components/utils/classNames";

export function Main({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <main className={cn("flex flex-col mx-auto max-w-6xl", className)}>
      {children}
    </main>
  );
}
