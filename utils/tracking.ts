export const trackPageView = (url: string) => {
  let ref = null;
  try {
    const urlParams = new URLSearchParams(window.location.search);
    ref = urlParams.get("ref");
  } catch (e) {}

  if (typeof window.Inngest === "undefined") {
    console.warn("Inngest is not initialized");
    return;
  }

  window.Inngest.event({
    name: "website/page.viewed",
    data: {
      first_touch: false,
      ref,
    },
  });

  // NOTE - Google Analytics is captured via Google Tag Manager's listening to the History API
};
