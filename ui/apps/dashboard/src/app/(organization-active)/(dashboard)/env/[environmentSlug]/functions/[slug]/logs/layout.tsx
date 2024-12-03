import DashboardRuns from './Runs';

type RunLayoutProps = {
  children: React.ReactNode;
  params: {
    slug: string;
  };
};

export default function RunLayout({ children, params }: RunLayoutProps) {
  return (
    <>
      <DashboardRuns params={params} />
      {children}
    </>
  );
}
