type LayoutProps = {
  children: React.ReactNode;
  slideOver: React.ReactNode;
};

export default function Layout({ children, slideOver }: LayoutProps) {
  return (
    <>
      {children}
      {slideOver}
    </>
  );
}
