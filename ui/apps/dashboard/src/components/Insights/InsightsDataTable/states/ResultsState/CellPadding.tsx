interface CellPaddingProps {
  children: React.ReactNode;
}

export function CellPadding({ children }: CellPaddingProps) {
  return (
    <div className="p-2.5 focus:outline-none" tabIndex={0}>
      {children}
    </div>
  );
}
