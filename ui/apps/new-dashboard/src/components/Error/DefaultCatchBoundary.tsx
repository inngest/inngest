import { Button } from "@inngest/components/Button/NewButton";
import { Error } from "@inngest/components/Error/Error";
import { Header } from "@inngest/components/Header/NewHeader";
import type { ErrorComponentProps } from "@tanstack/react-router";
import { rootRouteId, useMatch, useRouter } from "@tanstack/react-router";
import Layout from "../Layout/Layout";

export function DefaultCatchBoundary({ error }: ErrorComponentProps) {
  const router = useRouter();
  const isRoot = useMatch({
    strict: false,
    select: (state) => state.id === rootRouteId,
  });

  console.error(error.message);

  return (
    <Layout collapsed={false}>
      <Header breadcrumb={[{ text: "Error" }]} />
      <div className="flex flex-col justify-start items-start gap">
        <Error message={error.message} />

        <div className="flex gap-2 justif-start items-center flex-wrap mx-4">
          <Button
            kind="secondary"
            appearance="outlined"
            onClick={() => {
              router.invalidate();
            }}
            label="Try Again"
          />

          {isRoot ? (
            <Button
              to="/"
              kind="secondary"
              appearance="outlined"
              label="Home"
            />
          ) : (
            <Button
              kind="secondary"
              appearance="outlined"
              onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                e.preventDefault();
                window.history.back();
              }}
              label="Go Back"
            />
          )}
        </div>
      </div>
    </Layout>
  );
}
