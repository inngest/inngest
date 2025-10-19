'use client'

import { usePathname } from 'next/navigation'
import { Header } from '@inngest/components/Header/Header'

export default function Layout({ children }: React.PropsWithChildren) {
  const pathname = usePathname()

  return (
    <>
      <Header
        breadcrumb={[
          ...(pathname.includes('/choose-template')
            ? [{ text: 'Apps', href: '/apps' }, { text: 'Choose template' }]
            : []),
          ...(pathname.includes('/choose-framework')
            ? [{ text: 'Apps', href: '/apps' }, { text: 'Choose framework' }]
            : []),
        ]}
      />
      <div className="mx-auto flex w-full max-w-4xl flex-col px-6 pb-4 pt-16">
        {children}
      </div>
    </>
  )
}
