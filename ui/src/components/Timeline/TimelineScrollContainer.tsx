import { useAutoAnimate } from "@formkit/auto-animate/react";

export default function TimelineContainer({ children }) {
  const [animationRef] = useAutoAnimate<HTMLUListElement>({
    duration: 150,
  });

  return (
    <ul
      ref={animationRef}
      className="bg-slate-950/50 border-r border-slate-800/40 overflow-y-scroll relative py-4 pr-2.5 shrink-0 col-start-2 row-start-2 row-span-2 pt-[60px]"
    >
      {children}
    </ul>
  );
}
