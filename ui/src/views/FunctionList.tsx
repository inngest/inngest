import { useGetFunctionsQuery, type Function } from "../store/generated";
import FuncCard from "../components/Function/FuncCard";
import { BlankSlate } from "../components/Blank";
import { useAppDispatch } from "../store/hooks";
import { showDocs } from "../store/global";
import noResultsImg from "../../assets/images/no-results.png";
import { FunctionStatus } from "../utils/statusStyles";

export const FunctionList = () => {
  const dispatch = useAppDispatch();

  const { data, isFetching } = useGetFunctionsQuery();
  const functions = data?.functions || [];

  return (
    <div className="px-5 py-4 h-full flex flex-col overflow-y-scroll">
      <header className="mb-8">
        <h1 className="text-lg mb-2 text-slate-50">Functions</h1>
        <p>This is a list of all detected functions</p>
      </header>

      <div className="flex-1">
        {isFetching ? (
          <>Loading...</>
        ) : functions?.length ? (
          <div className="flex flex-col items-center basis-[36px] gap-4 max-w-xl">
            {functions.map((f, idx) => {
              const triggers = f?.triggers
                ?.map((t) => t.event || t.cron)
                .join(", ");
              const cleanUrl = new URL(f.url || "");
              cleanUrl.search = "";
              return (
                <FuncCard
                  key={f?.id || idx}
                  title={f?.name || "Missing function name"}
                  id={f?.id || "Invalid ID generated"}
                  status={FunctionStatus.Registered}
                  badge={triggers}
                  contextualBar={
                    <div className="text-3xs">{cleanUrl.toString()}</div>
                  }
                />
              );
            })}
          </div>
        ) : (
          <BlankSlate
            title="Inngest has not detected any functions"
            subtitle="Read our documentation to learn how to serve your functions"
            imageUrl={noResultsImg}
            button={{
              text: "Serving Functions",
              onClick: () => dispatch(showDocs("/sdk/serve")),
            }}
          />
        )}
      </div>
    </div>
  );
};
