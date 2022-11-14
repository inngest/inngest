export default function TimelineContainer({ children }) {
  return (
    <ul className="bg-slate-950 border-r border-slate-800 overflow-y-scroll relative py-4 pr-2.5 shrink-0 col-start-2 row-start-2 row-span-2 pt-[50px]">
      {children}
    </ul>
  )
}
