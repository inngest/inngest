import { useMemo, useState } from "react";
import { Button } from "@inngest/components/Button/NewButton";
import { Search } from "@inngest/components/Forms/Search";
import { StatusDot } from "@inngest/components/Status/StatusDot";
import useDebounce from "@inngest/components/hooks/useDebounce";

import Toaster from "@/components/Toast/Toaster";
import LoadingIcon from "@/components/Icons/LoadingIcon";
import { useEnvironments } from "@/queries";
import { EnvironmentType, type Environment } from "@/utils/environments";
import { BranchEnvironmentActions } from "./BranchEnvironmentActions";
import BranchEnvironmentListTable from "./BranchEnvironmentListTable";
import { CustomEnvironmentListTable } from "./CustomEnvironmentListTable";
import { EnvironmentsStatusSelector } from "./EnvironmentsStatusSelector";
import { EnvKeysDropdownButton } from "./row-actions/EnvKeysDropdownButton";
import { EnvViewButton } from "./row-actions/EnvViewButton";

export default function Environments() {
  const [{ data: envs = [], fetching }] = useEnvironments();

  const [filterStatus, setFilterStatus] = useState<"active" | "archived">(
    "active",
  );

  const [searchInput, setSearchInput] = useState<string>("");
  const [searchParam, setSearchParam] = useState<string>("");
  const debouncedSearch = useDebounce(() => {
    setSearchParam(searchInput);
  }, 400);

  const branchParent = envs.find(
    (env) => env.type === EnvironmentType.BranchParent,
  );

  const branchEnvsData = useMemo(() => {
    return filterEnvironments(
      EnvironmentType.BranchChild,
      searchParam,
      filterStatus,
      envs,
    );
  }, [searchParam, envs, filterStatus]);

  const customEnvsData = useMemo(() => {
    return filterEnvironments(
      EnvironmentType.Test,
      searchParam,
      filterStatus,
      envs,
    );
  }, [searchParam, envs, filterStatus]);

  const prodEnvsData = useMemo(() => {
    return filterEnvironments(
      EnvironmentType.Production,
      searchParam,
      filterStatus,
      envs,
    );
  }, [searchParam, envs, filterStatus]);

  if (fetching) {
    return (
      <div className="my-16 flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const isMultiProd = prodEnvsData.total > 1;

  return (
    <>
      <div className="mx-auto w-full max-w-[860px] px-12 py-16">
        {!isMultiProd && (
          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-2">
              <div className="flex w-full items-center justify-between">
                <h2 className="text-xl font-medium">Production environment</h2>
              </div>

              <p className="text-muted max-w-[400px] text-sm">
                This is where you&apos;ll deploy all of your production apps.
              </p>
            </div>

            <div className="border-muted rounded-md border">
              <div className="border-l-primary-moderate flex min-w-0 items-center justify-between overflow-x-auto rounded-[4px] border-l-4 px-4 py-3">
                <h3 className="flex flex-shrink-0 items-center gap-2 text-sm font-medium tracking-wide">
                  <StatusDot status="ACTIVE" size="small" />
                  Production
                </h3>
                <div className="flex flex-shrink-0 items-center gap-2 pl-2">
                  <EnvViewButton env={{ slug: "production" }} />
                  <EnvKeysDropdownButton env={{ slug: "production" }} />
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="mb-2 flex flex-col gap-3">
          <div className="border-subtle mt-8 flex w-full items-center justify-between border-t pt-8">
            <h2 className="mt-1 text-xl font-medium">Other environments</h2>
          </div>
          <div className="flex w-full flex-wrap gap-3">
            <EnvironmentsStatusSelector
              archived={filterStatus === "archived"}
              onChange={(archived: boolean) => {
                setFilterStatus(archived ? "archived" : "active");
              }}
            />
            <div className="min-w-[200px] flex-auto">
              <Search
                className="h-[34px] w-full"
                name="search-other-envs"
                onUpdate={(value) => {
                  setSearchInput(value);
                  debouncedSearch();
                }}
                placeholder="Search environments"
                value={searchInput}
                inngestSize="base"
              />
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <div className="flex flex-col gap-3 pt-6">
            {isMultiProd && (
              <>
                <div className="flex w-full flex-wrap items-center justify-between gap-3">
                  <h2 className="text-md font-medium">
                    Production environments
                  </h2>
                </div>
                <div className="border-subtle overflow-hidden rounded-md border">
                  <CustomEnvironmentListTable
                    envs={prodEnvsData.filtered}
                    paginationKey={getPaginationKey(filterStatus, searchParam)}
                    unfilteredEnvsCount={prodEnvsData.total}
                  />
                </div>
              </>
            )}

            <div className="flex w-full flex-wrap items-center justify-between gap-3">
              <h2 className="text-md font-medium">Custom environments</h2>
              <Button
                className="text-sm"
                href="/create-environment"
                kind="primary"
                label="Create custom environment"
              />
            </div>
            <div className="border-subtle overflow-hidden rounded-md border">
              <CustomEnvironmentListTable
                envs={customEnvsData.filtered}
                paginationKey={getPaginationKey(filterStatus, searchParam)}
                unfilteredEnvsCount={customEnvsData.total}
              />
            </div>
          </div>

          {Boolean(branchParent) && (
            <div className="flex flex-col gap-3">
              <div className="flex w-full flex-wrap items-center justify-between gap-3">
                <h2 className="text-md font-medium">Branch environments</h2>
                <div className="flex items-center gap-2">
                  <BranchEnvironmentActions
                    branchParent={branchParent as Environment}
                  />
                </div>
              </div>
              <div className="border-subtle overflow-hidden rounded-md border">
                <BranchEnvironmentListTable
                  envs={branchEnvsData.filtered}
                  paginationKey={getPaginationKey(filterStatus, searchParam)}
                  unfilteredEnvsCount={branchEnvsData.total}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      <Toaster />
    </>
  );
}

// This is used to reset to page 1 when the filter or search changes.
function getPaginationKey(
  filterStatus: "active" | "archived",
  searchParam: string,
) {
  return `${filterStatus}:${searchParam}`;
}

function filterEnvironments(
  type: EnvironmentType,
  searchParam: string,
  filterStatus: "active" | "archived",
  envs: Environment[],
) {
  const filtered: Environment[] = [];
  let total = 0;

  for (const env of envs) {
    if (env.type !== type) continue;

    total++;

    const matchesSearch =
      searchParam === "" ||
      env.name.toLowerCase().includes(searchParam.toLowerCase());
    const matchesStatus =
      filterStatus === "archived" ? env.isArchived : !env.isArchived;

    if (matchesSearch && matchesStatus) filtered.push(env);
  }

  return { filtered, total };
}
