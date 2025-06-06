import FunctionTable from '@/app/(dashboard)/functions/FunctionTable';

type FunctionLayoutProps = {
  children: React.ReactNode;
};

export default function FunctionLayout({ children }: FunctionLayoutProps) {
  // TODO should we call it FunctionList
  return (
    <>
      <FunctionTable />
      {children}
    </>
  );
}
