import type { ComponentChildren } from 'preact'

type Props = {
  children?: ComponentChildren
}

export default function Sidebar(props: Props) {
  return (
    <div className="h-full bg-slate-950/50 border-r border-slate-800/60 row-span-2 col-start-1 flex-col items-start pt-2 ">
      {props.children}
    </div>
  )
}
