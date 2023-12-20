import AppNavigation from '@/components/Navigation/AppNavigation';
import Toaster from '@/components/Toaster';

type RootLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: React.ReactNode;
};

export default function RootLayout({ params, children }: RootLayoutProps) {
  const environmentSlug = params.environmentSlug ?? 'production';
  return (
    <>
      <div className="isolate flex h-full flex-col">
        <AppNavigation environmentSlug={environmentSlug} />
        {children}
      </div>
      <Toaster
        toastOptions={{
          // Ensure that the toast is clickable when there are overlays/modals
          className: 'pointer-events-auto',
        }}
      />
    </>
  );
}
