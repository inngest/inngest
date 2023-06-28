type Props = {
  children?: React.ReactNode;
};

export default function Navbar(props: Props) {
  return (
    <nav className=" bg-slate-950/50 flex items-center gap-3 border-l border-slate-800 pl-3">
      {props.children}
    </nav>
  );
}
