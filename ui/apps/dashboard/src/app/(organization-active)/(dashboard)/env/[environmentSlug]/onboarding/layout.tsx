import { Header } from '@inngest/components/Header/Header';

import { pathCreator } from '@/utils/urls';

export default function Layout({ children }: React.PropsWithChildren) {
  return (
    <>
      <Header breadcrumb={[{ text: 'Getting started', href: pathCreator.onboarding() }]} />
      {children}
    </>
  );
}
