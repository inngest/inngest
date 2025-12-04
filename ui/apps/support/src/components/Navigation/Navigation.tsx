// import { Suspense } from "react";
// import { Skeleton } from "@inngest/components/Skeleton/Skeleton";

// import type { Environment as EnvType } from "@/utils/environments";
// import Environments from "./Environments";
// import KeysMenu from "./KeysMenu";
// import Manage from "./Manage";
// import Monitor from "./Monitor";

export type NavProps = {
  collapsed: boolean;
};

export default function Navigation({ collapsed }: NavProps) {
  return (
    <div className={`text-basis mx-4 mt-4 flex h-full flex-col`}>
      Navigation
    </div>
  );
}
