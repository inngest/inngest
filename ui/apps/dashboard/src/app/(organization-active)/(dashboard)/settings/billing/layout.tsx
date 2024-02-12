export default async function BillingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="overflow-y-scroll">
      <div className="mx-auto max-w-screen-xl">
        <header className="border-b border-slate-200 py-6">
          <h1 className="text-2xl font-semibold">Billing</h1>
        </header>
        {children}
      </div>
    </div>
  );
}
