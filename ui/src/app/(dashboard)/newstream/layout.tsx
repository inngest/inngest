type StreamLayoutProps = {
  children: React.ReactNode;
  slideOver: React.ReactNode;
};

export default function StreamLayout({ children, slideOver }: StreamLayoutProps) {
  return (
    <>
      {children}
      {slideOver}
    </>
  );
}
