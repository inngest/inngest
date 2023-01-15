type HeadingProps = {
  level: 1 | 2 | 3 | 4 | 5;
  children: React.ReactNode;
  id: string;
  tag?: string;
  label?: string;
  anchor?: boolean;
};

function Heading({
  children,
  level = 2,
  id,
  tag,
  label,
  anchor = true,
}: HeadingProps) {
  let Component: "h1" | "h2" | "h3" | "h4" | "h5" = `h${level}`;

  return (
    <>
      <Component id={id} className="scroll-mt-32">
        {children}
      </Component>
    </>
  );
}

export const h2: React.FC<any> = function H2(props) {
  return <Heading level={2} {...props} />;
};
export const h3: React.FC<any> = function H2(props) {
  return <Heading level={3} {...props} />;
};
export const h4: React.FC<any> = function H2(props) {
  return <Heading level={3} {...props} />;
};
