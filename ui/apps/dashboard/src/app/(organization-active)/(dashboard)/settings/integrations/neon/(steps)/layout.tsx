import { StepsProvider } from '@/components/PostgresIntegration/Context';

export default function Layout({ children }: React.PropsWithChildren) {
  return <StepsProvider>{children}</StepsProvider>;
}
