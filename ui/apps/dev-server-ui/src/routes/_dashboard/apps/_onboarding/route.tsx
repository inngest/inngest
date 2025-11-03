import { Header } from '@inngest/components/Header/NewHeader';
import { createFileRoute, Outlet, useLocation } from '@tanstack/react-router';

export const Route = createFileRoute('/_dashboard/apps/_onboarding')({
  component: OnboardingComponent,
});

function OnboardingComponent() {
  const location = useLocation();
  return (
    <>
      <Header
        breadcrumb={[
          ...(location.pathname.includes('/choose-template')
            ? [{ text: 'Apps', href: '/apps' }, { text: 'Choose template' }]
            : []),
          ...(location.pathname.includes('/choose-framework')
            ? [{ text: 'Apps', href: '/apps' }, { text: 'Choose framework' }]
            : []),
        ]}
      />
      <div className="mx-auto flex w-full max-w-4xl flex-col px-6 pb-4 pt-16">
        <Outlet />
      </div>
    </>
  );
}
