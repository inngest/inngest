import { useState, useEffect } from "react";

type Indicator = "none" | "minor" | "major" | "critical";
type StatusPageStatusResponse = {
  page: {
    id: string;
    name: string;
    url: string;
    updated_at: string;
  };
  status: {
    description: string;
    indicator: Indicator;
  };
};

type Status = {
  url: string;
  description: string;
  indicator: Indicator;
  updated_at: string;
};

const fetchStatus = async (): Promise<StatusPageStatusResponse> => {
  return await fetch("https://inngest.statuspage.io/api/v2/status.json").then(
    (r) => r.json()
  );
};

const useStatus = (): Status => {
  const [status, setStatus] = useState<Status>({
    url: "http://status.inngest.com", // Not https
    description: "Fetching status...",
    indicator: "none",
    updated_at: "",
  });
  useEffect(() => {
    (async function () {
      const res = await fetchStatus();
      setStatus({
        ...res.status,
        updated_at: res.page.updated_at,
        url: res.page.url,
      });
    })();
  }, []);
  return status;
};

// We use hex colors b/c tailwind only includes what is initially rendered
const statusColor: { [K in Indicator]: string } = {
  none: "#22c55e", // green-500
  minor: "#fde047", // yellow-300
  major: "#f97316", // orange-500
  critical: "#dc2626", // red-600
};

export function StatusIcon({ className = "" }: { className?: string }) {
  const status = useStatus();
  return (
    <span className={`${className} inline-flex items-center justify-center`}>
      <span
        className={`inline-flex m-auto w-2 h-2 rounded-full`}
        style={{ backgroundColor: statusColor[status.indicator] }}
        title={`${status.description} - Status updated at ${status.updated_at}`}
      ></span>
    </span>
  );
}

export default function StatusWidget({
  className = "",
}: {
  className?: string;
}) {
  const status = useStatus();
  return (
    <a
      href={status.url}
      target="_blank"
      rel="noopener noreferrer"
      className={`${className} text-slate-200 font-medium bg-slate-900 hover:bg-slate-800 transition-all rounded text-sm px-4 py-2 inline-flex items-center`}
      title={`Status updated at ${status.updated_at}`}
    >
      <span
        className={`inline-flex w-2 h-2 mr-2 rounded-full`}
        style={{ backgroundColor: statusColor[status.indicator] }}
      ></span>
      {status.description}
    </a>
  );
}
