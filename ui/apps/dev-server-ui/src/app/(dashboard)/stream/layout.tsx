import Link from 'next/link';
import { Button } from '@inngest/components/Button/Button';
import { Header } from '@inngest/components/Header/Header';
import { Info } from '@inngest/components/Info/Info';

import SendEventButton from '@/components/Event/SendEventButton';
import Stream from './Stream';

type StreamLayoutProps = {
  children: React.ReactNode;
};

export default function StreamLayout({ children }: StreamLayoutProps) {
  return (
    <>
      <Stream />
      {children}
    </>
  );
}
