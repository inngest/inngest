import FunctionList from '@/app/(dashboard)/functions/FunctionList'

type FunctionLayoutProps = {
  children: React.ReactNode
}

export default function FunctionLayout({ children }: FunctionLayoutProps) {
  return (
    <>
      <FunctionList />
      {children}
    </>
  )
}
