type Props = {
  children?: React.ReactNode;
};

export default function Sidebar(props: Props) {
  return (
    <div className="h-full bg-slate-950/50 border-r border-slate-800/40 row-span-2 col-start-1 flex-col items-start pt-2 ">
      {props.children}
    </div>
  );
}
