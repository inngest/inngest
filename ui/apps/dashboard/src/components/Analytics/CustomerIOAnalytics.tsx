import { useOrganization, useUser } from '@clerk/tanstack-react-start';
import { useEffect } from 'react';

const WRITE_KEY = import.meta.env.VITE_CUSTOMERIO_WRITE_KEY;

function loadSnippet(writeKey: string) {
  const i = 'cioanalytics';
  const analytics = ((window as any)[i] = (window as any)[i] || []);
  if (analytics.initialize) return;
  if (analytics.invoked) return;
  analytics.invoked = true;
  analytics.methods = [
    'trackSubmit',
    'trackClick',
    'trackLink',
    'trackForm',
    'pageview',
    'identify',
    'reset',
    'group',
    'track',
    'ready',
    'alias',
    'debug',
    'page',
    'once',
    'off',
    'on',
    'addSourceMiddleware',
    'addIntegrationMiddleware',
    'setAnonymousId',
    'addDestinationMiddleware',
  ];
  analytics.factory = function (method: string) {
    return function (...args: any[]) {
      args.unshift(method);
      analytics.push(args);
      return analytics;
    };
  };
  for (const method of analytics.methods) {
    analytics[method] = analytics.factory(method);
  }
  analytics.load = function (key: string, options?: any) {
    const script = document.createElement('script');
    script.type = 'text/javascript';
    script.async = true;
    script.setAttribute('data-global-customerio-analytics-key', i);
    script.src =
      'https://cdp.customer.io/v1/analytics-js/snippet/' +
      key +
      '/analytics.min.js';
    const first = document.getElementsByTagName('script')[0];
    first?.parentNode?.insertBefore(script, first);
    analytics._writeKey = key;
    analytics._loadOptions = options;
  };
  analytics.SNIPPET_VERSION = '4.15.3';
  analytics.load(writeKey);
}

function getCioAnalytics(): any | undefined {
  return (window as any).cioanalytics;
}

export default function CustomerIOAnalytics() {
  useEffect(() => {
    if (!import.meta.env.PROD || !WRITE_KEY) return;
    loadSnippet(WRITE_KEY);
  }, []);

  const { user, isSignedIn } = useUser();
  const { organization } = useOrganization();

  useEffect(() => {
    if (!import.meta.env.PROD || !WRITE_KEY) return;
    if (!isSignedIn || !organization) return;

    const cio = getCioAnalytics();
    if (!cio) return;

    cio.identify(user.externalId, {
      email: user.primaryEmailAddress?.emailAddress,
      name: user.fullName,
      clerk_user_id: user.id,
    });
    if (organization.publicMetadata.accountID) {
      cio.group(organization.publicMetadata.accountID, {
        name: organization.name,
      });
    }
  }, [isSignedIn, organization, user]);

  return null;
}
