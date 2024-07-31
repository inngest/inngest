import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import FunctionLayout from './newLayout';
import OldFunctionLayout from './oldLayout';

type FunctionLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default async function Layout({ children, params }: FunctionLayoutProps) {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <FunctionLayout params={params}>{children}</FunctionLayout>
  ) : (
    <OldFunctionLayout params={params}>{children}</OldFunctionLayout>
  );
}
