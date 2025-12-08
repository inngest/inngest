"use client";

import dynamic from "next/dynamic";

import EventsPage from "@/components/Events/EventsPage";

const EventsFeedback = dynamic(
  () => import("@/components/Surveys/EventsFeedback"),
  {
    ssr: false,
  },
);

export default function Page({
  params: { environmentSlug: envSlug },
}: {
  params: { environmentSlug: string };
}) {
  return (
    <>
      <EventsPage environmentSlug={envSlug} showHeader />
      <EventsFeedback />
    </>
  );
}
