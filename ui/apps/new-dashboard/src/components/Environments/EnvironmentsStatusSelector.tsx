"use client";

import { Select } from "@inngest/components/Select/Select";
import { StatusDot } from "@inngest/components/Status/StatusDot";

type EnvironmentsStatusSelectorProps = {
  archived: boolean;
  onChange: (archived: boolean) => void;
};

const ACTIVE_OPTION = { id: "active", name: "Active environments" };
const ARCHIVED_OPTION = { id: "archived", name: "Archived environments" };

export function EnvironmentsStatusSelector({
  archived,
  onChange,
}: EnvironmentsStatusSelectorProps) {
  return (
    <Select
      onChange={(value) => onChange(value.id === "archived")}
      isLabelVisible={false}
      label="Select environment status"
      multiple={false}
      value={archived ? ARCHIVED_OPTION : ACTIVE_OPTION}
    >
      <Select.Button className="w-[200px] shrink-0">
        <div className="mr-1 flex flex-row items-center gap-2 overflow-hidden whitespace-nowrap">
          <StatusDot status={archived ? "ARCHIVED" : "ACTIVE"} size="small" />
          {archived ? "Archived environments" : "Active environments"}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={ACTIVE_OPTION.id} option={ACTIVE_OPTION}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ACTIVE" size="small" />
            {ACTIVE_OPTION.name}
          </div>
        </Select.Option>

        <Select.Option key={ARCHIVED_OPTION.id} option={ARCHIVED_OPTION}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ARCHIVED" size="small" />
            {ARCHIVED_OPTION.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
}
