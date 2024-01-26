import RunsPage from './Runs';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    slug: string;
  };
};

export default function RunLayout({ children, params }: RunLayoutProps) {
  return (
    <>
      <RunsPage params={params} />
      {children}
    </>
  );
}
