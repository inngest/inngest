"use client";

import { useCallback, useRef, useState } from "react";
import { Search } from "@inngest/components/Forms/Search";
import { InfiniteScrollTrigger } from "@inngest/components/InfiniteScrollTrigger/InfiniteScrollTrigger";
import { Pill } from "@inngest/components/Pill/Pill";
import { SchemaViewer } from "@inngest/components/SchemaViewer/SchemaViewer";
import type { ValueNode } from "@inngest/components/SchemaViewer/types";

import { SchemaExplorerSwitcher } from "./SchemaExplorerSwitcher";
import { useSchemas } from "./SchemasContext/SchemasContext";
import { useSchemasInUse } from "./useSchemasInUse";

export function SchemaExplorer() {
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const {
    entries,
    error,
    hasFetchedMax,
    hasNextPage,
    fetchNextPage,
    isLoading,
    isFetchingNextPage,
  } = useSchemas({
    search,
  });

  const { schemasInUse } = useSchemasInUse();

  const renderSharedAdornment = useCallback((node: ValueNode) => {
    if (node.path !== "events") return null;
    return (
      <Pill
        appearance="outlined"
        className="border-subtle text-subtle"
        kind="secondary"
      >
        Shared schema
      </Pill>
    );
  }, []);

  const renderEntry = useCallback(
    (entry: (typeof entries)[number], preventExpand: boolean = false) => {
      const isCommonEventSchema = entry.key === "common:events";

      return (
        <SchemaViewer
          key={entry.key}
          computeType={
            isCommonEventSchema ? computeSharedEventSchemaType : undefined
          }
          defaultExpandedPaths={
            preventExpand
              ? undefined
              : isCommonEventSchema
              ? ["events"]
              : undefined
          }
          node={entry.node}
          renderAdornment={
            isCommonEventSchema ? renderSharedAdornment : undefined
          }
        />
      );
    },
    [renderSharedAdornment],
  );

  return (
    <div
      className="flex h-full w-full flex-col gap-3 overflow-auto p-4"
      ref={containerRef}
    >
      <>
        {schemasInUse.length > 0 && (
          <div className="mb-3 flex flex-col gap-2">
            <div className="text-light text-xs font-medium uppercase">
              Schemas in Use
            </div>
            <div className="flex flex-col gap-1">
              {schemasInUse.map((schema) => renderEntry(schema, true))}
            </div>
          </div>
        )}
        <div className="text-light text-xs font-medium uppercase">
          All Schemas
        </div>
        <Search
          inngestSize="base"
          onUpdate={setSearch}
          placeholder="Search event type"
          value={search}
        />
      </>
      <div className="flex flex-col gap-1">
        <SchemaExplorerSwitcher
          entries={entries}
          error={error}
          isLoading={isLoading}
          isFetchingNextPage={isFetchingNextPage}
          hasFetchedMax={hasFetchedMax}
          hasNextPage={hasNextPage}
          fetchNextPage={fetchNextPage}
          renderEntry={renderEntry}
        />
        <InfiniteScrollTrigger
          onIntersect={fetchNextPage}
          hasMore={hasNextPage && !error && !hasFetchedMax}
          isLoading={isLoading || isFetchingNextPage}
          root={containerRef.current}
          rootMargin="50px"
        />
      </div>
    </div>
  );
}

function computeSharedEventSchemaType(
  node: ValueNode,
  baseLabel: string,
): string {
  if (node.path === "events.data" && baseLabel === "string") return "JSON";
  return baseLabel;
}
