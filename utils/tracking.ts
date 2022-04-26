export const trackPageView = (url: string) => {
  let ref = null;
  try {
    const urlParams = new URLSearchParams(window.location.search);
    ref = urlParams.get("ref");
  } catch (e) {}

  window.Inngest.event({
    name: "website/page.viewed",
    data: {
      first_touch: false,
      ref,
    },
  });

  // NOTE - Google Analytics is captured via Google Tag Manager's listening to the History API
};
