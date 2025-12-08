"use client";

import { Debugger } from "@inngest/components/Debugger/Debugger";
import { Header } from "@inngest/components/Header/Header";

export default function Page({ params }: { params: { functionSlug: string } }) {
  const functionSlug = decodeURIComponent(params.functionSlug);

  return (
    <>
      <Header
        breadcrumb={[
          { text: "Runs" },
          { text: functionSlug },
          { text: "Debug" },
        ]}
        action={<div className="flex flex-row items-center gap-x-1"></div>}
      />
      <Debugger functionSlug={functionSlug} />
    </>
  );
}
