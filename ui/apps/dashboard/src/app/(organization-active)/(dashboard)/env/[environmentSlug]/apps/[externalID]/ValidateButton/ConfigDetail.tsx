import { isRecord } from '@inngest/components/utils/object';

import {
  SdkMode,
  SecretCheck,
  type AppCheckFieldBoolean,
  type AppCheckFieldString,
  type AppCheckResult,
} from '@/gql/graphql';

type Props = {
  data: AppCheckResult;
};

export function ConfigDetail({ data }: Props) {
  return (
    <table className="w-full">
      <ConfigRow label="API origin" value={data.apiOrigin} />
      <ConfigRow label="App ID" value={data.appID} />
      <ConfigRow label="Environment" value={data.env} />
      <ConfigRow label="Event API origin" value={data.eventAPIOrigin} />
      <ConfigRow label="Framework" value={data.framework} />
      <ConfigRow label="Event key" value={data.eventKeyStatus} />
      <ConfigRow label="Signing key" value={data.signingKeyStatus} />
      <ConfigRow label="Signing key fallback" value={data.signingKeyFallbackStatus} />
      <ConfigRow label="Mode" value={data.mode} />
      <ConfigRow label="SDK language" value={data.sdkLanguage} />
      <ConfigRow label="SDK version" value={data.sdkVersion} />
      <ConfigRow label="Serve origin" value={data.serveOrigin} />
      <ConfigRow label="Serve path" value={data.servePath} />
      <ConfigRow label="Extra" value={data.extra} />
    </table>
  );
}

function ConfigRow({
  label,
  value,
}: {
  label: string;
  value:
    | AppCheckFieldBoolean
    | AppCheckFieldString
    | SdkMode
    | SecretCheck
    | Record<string, unknown>
    | boolean
    | null;
}) {
  let text: React.ReactNode = '';
  if (value === null) {
    text = 'UNKNOWN';
  } else if (typeof value === 'boolean') {
    text = value ? 'Yes' : 'No';
  } else if (isAppCheckFieldBoolean(value)) {
    if (value.value === null) {
      text = '';
    } else {
      text = value.value ? 'Yes' : 'No';
    }
  } else if (isAppCheckFieldString(value)) {
    if (value.value === null) {
      text = '';
    } else {
      text = value.value;
    }
  } else if (isSDKMode(value)) {
    text = value;
  } else if (isSecretCheck(value)) {
    text = value;
  } else if (isRecord(value)) {
    text = JSON.stringify(value, null, 2);
  }

  return (
    <tr className="border-subtle text-basis border-b text-sm">
      <td className="px-3 py-1.5 align-top">{label}</td>
      <td className="py-1.5">
        <pre>{text}</pre>
      </td>
    </tr>
  );
}

function isAppCheckFieldBoolean(value: unknown): value is AppCheckFieldBoolean {
  return (
    typeof value === 'object' &&
    value !== null &&
    '__typename' in value &&
    value.__typename === 'AppCheckFieldBoolean'
  );
}

function isAppCheckFieldString(value: unknown): value is AppCheckFieldString {
  return (
    typeof value === 'object' &&
    value !== null &&
    '__typename' in value &&
    value.__typename === 'AppCheckFieldString'
  );
}

function isSDKMode(value: unknown): value is SdkMode {
  return Object.values(SdkMode).includes(value as SdkMode);
}

function isSecretCheck(value: unknown): value is SecretCheck {
  return Object.values(SecretCheck).includes(value as SecretCheck);
}
