import { useEffect, useState } from "react";

export type For = "events" | "actions";

export type Method = {
  method: "API" | "OAuth" | "Webhook";
  secrets: string[];
  for: For[];
  automated: boolean;
  description?: string;
  transform?: string;
};

export type Integration = {
  name: string;
  logo: { url: string };
  tags: string[];
  service: string;
  methods: Array<Method>;
  help: Array<{ title: string; body: string }>;
};

export const fetchIntegrations = async (setter: (a: any) => void) => {
  try {
    const result = await fetch(
      "https://api.inngest.com/v1/public/integrations"
    );
    setter(await result.json());
  } catch (e) {}
};

export const useIntegration = async (
  name: string,
  setter: (a: any) => void
) => {
  const getter = async () => {
    if (!name) {
      return;
    }
    try {
      const result = await fetch(
        `https://api.inngest.com/v1/public/integrations/${name}`
      );
      setter(await result.json());
    } catch (e) {}
  };
  useEffect(() => {
    getter();
  }, [name]);
};

export const useIntegrations = () => {
  const [json, setJSON] = useState<{ [name: string]: Integration }>({});
  useEffect(() => {
    fetchIntegrations(setJSON);
  }, []);
  return json;
};
