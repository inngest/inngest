import Stream from './Stream';

type StreamLayoutProps = {
  children: React.ReactNode;
};

export default function StreamLayout({ children }: StreamLayoutProps) {
  return (
    <>
      <Stream />
      {children}
    </>
  );
}
