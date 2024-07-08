type Props = React.PropsWithChildren<{
  description?: React.ReactNode;
  name: string;
  optional?: boolean;
}>;

export function Label({ children, description, name, optional }: Props) {
  return (
    <>
      <label htmlFor={name}>
        {children}
        {optional && <span className="text-subtle text-sm"> (optional)</span>}
      </label>

      {description && <p className="text-subtle text-sm">{description}</p>}
    </>
  );
}
