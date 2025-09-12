interface CellPaddingProps {
  children: React.ReactNode;
}

export function CellPadding({ children }: CellPaddingProps) {
  return <div className="p-2.5">{children}</div>;
}
