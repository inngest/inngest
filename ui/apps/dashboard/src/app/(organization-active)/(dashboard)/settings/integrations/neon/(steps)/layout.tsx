import { StepsProvider } from './Context';

export default function Layout({ children }: React.PropsWithChildren) {
  return <StepsProvider>{children}</StepsProvider>;
}
