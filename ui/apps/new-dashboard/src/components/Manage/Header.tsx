import { Header } from "@inngest/components/Header/Header";

import CreateKeyButton from "@/components/Manage/CreateKeyButton";
import { EventKeyInfo } from "./EventKeyInfo";
import { SigningKeyInfo } from "./SigningKeyInfo";
import { WebhookInfo } from "./WebhookInfo";
import { useLocation } from "@tanstack/react-router";

export const ManageHeader = () => {
  const location = useLocation();
  const pathname = location.pathname;
  return (
    <Header
      breadcrumb={[
        ...(pathname.includes("/webhooks") ? [{ text: "Webhooks" }] : []),
        ...(pathname.includes("/keys") ? [{ text: "Event Keys" }] : []),
        ...(pathname.includes("/signing-key") ? [{ text: "Signing Key" }] : []),
      ]}
      infoIcon={
        <>
          {pathname.includes("/webhooks") && <WebhookInfo />}
          {pathname.includes("/keys") && <EventKeyInfo />}
          {pathname.includes("/signing-key") && <SigningKeyInfo />}
        </>
      }
      action={<CreateKeyButton />}
    />
  );
};
