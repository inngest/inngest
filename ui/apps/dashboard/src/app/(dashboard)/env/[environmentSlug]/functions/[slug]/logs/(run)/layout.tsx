import { SlideOver } from '@inngest/components/SlideOver';

type RunLayoutProps = {
  children: React.ReactNode;
};

export default function RunLayout({ children }: RunLayoutProps) {
  return <SlideOver size="large">{children}</SlideOver>;
}
