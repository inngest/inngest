import { useNavigate } from "@tanstack/react-router";
import { Select } from "@inngest/components/Select/Select";
import { StatusDot } from "@inngest/components/Status/StatusDot";

export const StatusMenu = ({
  envSlug,
  archived,
}: {
  envSlug: string;
  archived: boolean;
}) => {
  const navigate = useNavigate();
  const activeOption = { id: "active", name: "Active apps" };
  const archivedOption = { id: "archived", name: "Archived apps" };

  const handleChange = (
    option: typeof activeOption | typeof archivedOption,
  ) => {
    navigate({
      to: "/env/$envSlug/apps",
      params: { envSlug },
      search: { archived: option.id === "archived" ? "true" : undefined },
      replace: true,
    });
  };

  return (
    <Select
      onChange={handleChange}
      isLabelVisible={false}
      label="Select app status"
      multiple={false}
      value={archived ? archivedOption : activeOption}
      className="mb-5"
    >
      <Select.Button className="w-[132px]">
        <div className="flex flex-row items-center gap-2">
          <StatusDot status={archived ? "ARCHIVED" : "ACTIVE"} size="small" />
          {archived ? "Archived" : "Active"}
        </div>
      </Select.Button>
      <Select.Options>
        <Select.Option key={activeOption.id} option={activeOption}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ACTIVE" size="small" />
            {activeOption.name}
          </div>
        </Select.Option>
        <Select.Option key={archivedOption.id} option={archivedOption}>
          <div className="flex flex-row items-center gap-2">
            <StatusDot status="ARCHIVED" size="small" />
            {archivedOption.name}
          </div>
        </Select.Option>
      </Select.Options>
    </Select>
  );
};
