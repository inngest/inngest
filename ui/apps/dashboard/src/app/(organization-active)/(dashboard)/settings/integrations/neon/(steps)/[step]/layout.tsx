import PageHeader from '@/components/PostgresIntegration/PageHeader';
import StepsMenu from '@/components/PostgresIntegration/StepsMenu';

export default async function Layout(props: React.PropsWithChildren<{ params: { step: string } }>) {
  const params = await props.params;

  const { step } = params;

  const { children } = props;

  return (
    <div className="text-subtle my-12 grid grid-cols-3">
      <main className="col-span-2 mx-20">
        <PageHeader step={step} integration="neon" />
        {children}
      </main>
      <StepsMenu step={step} />
    </div>
  );
}
