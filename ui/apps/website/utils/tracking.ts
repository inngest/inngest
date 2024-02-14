export const trackPageView = (url: string) => {
  let ref = null;
  try {
    const urlParams = new URLSearchParams(window.location.search);
    ref = urlParams.get('ref');
  } catch (e) {}

  if (typeof window.Inngest === 'undefined') {
    console.warn('Inngest is not initialized');
    return;
  }

  window.Inngest.event({
    name: 'website/page.viewed',
    data: {
      first_touch: false,
      ref,
    },
    v: '2022-12-27.1',
  });

  // NOTE - Google Analytics is captured via Google Tag Manager's listening to the History API
};

export const trackDemoView = () => {
  let ref = null;
  try {
    const urlParams = new URLSearchParams(window.location.search);
    ref = urlParams.get('ref');
  } catch (e) {}

  if (typeof window.Inngest === 'undefined') {
    console.warn('Inngest is not initialized');
    return;
  }

  window.Inngest.event({
    name: 'website/demo.viewed',
    data: {
      ref,
    },
  });
};
