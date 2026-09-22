import {
  maxSecretImportCount,
  validateSecretName,
  validateSecretValue,
  type SecretInput,
} from './validation';

type ParseResult =
  | { ok: true; entries: SecretInput[] }
  | { ok: false; error: string };

/** Parse pasted assignments without expansion, evaluation, or silently dropping lines. */
export function parseEnv(source: string): ParseResult {
  if (new TextEncoder().encode(source).length > 1024 * 1024) {
    return { ok: false, error: 'Paste up to 1 MiB at a time.' };
  }
  const lines = source
    .replace(/^\uFEFF/, '')
    .replace(/\r\n?/g, '\n')
    .split('\n');
  const entries: SecretInput[] = [];
  for (let index = 0; index < lines.length; index++) {
    const lineNumber = index + 1;
    const line = lines[index].trimStart();
    if (!line || line.startsWith('#')) continue;
    const match = /^(?:export[\t ]+)?([^=]+)=(.*)$/.exec(line);
    if (!match)
      return { ok: false, error: `Line ${lineNumber}: expected NAME=value.` };
    const name = match[1].trim();
    let value = match[2].trimStart();
    const quote = value[0];
    if (quote === '"' || quote === "'" || quote === '`') {
      value = value.slice(1);
      let end = -1;
      // A quoted value can span lines; escaped quotes do not end the value.
      for (let offset = 0; ; offset++) {
        if (offset >= value.length) {
          if (++index >= lines.length) {
            return {
              ok: false,
              error: `Line ${lineNumber}: close the quoted value.`,
            };
          }
          value += '\n' + lines[index];
        }
        if (value[offset] === '\\' && offset + 1 < value.length) {
          offset++;
        } else if (value[offset] === quote) {
          end = offset;
          break;
        }
      }
      const rest = value.slice(end + 1).trim();
      if (rest && !rest.startsWith('#')) {
        return {
          ok: false,
          error: `Line ${lineNumber}: unexpected text after the quoted value.`,
        };
      }
      value = value.slice(0, end);
      if (quote === '"')
        value = value.replace(/\\n/g, '\n').replace(/\\r/g, '\r');
    } else {
      value = value.split('#', 1)[0].trim();
    }
    const error = validateSecretName(name) ?? validateSecretValue(value);
    if (error) return { ok: false, error: `Line ${lineNumber}: ${error}` };
    entries.push({ name, value });
    if (entries.length > maxSecretImportCount) {
      return {
        ok: false,
        error: `Paste up to ${maxSecretImportCount} secrets at a time.`,
      };
    }
  }
  return entries.length
    ? { ok: true, entries }
    : { ok: false, error: 'Paste one or more NAME=value assignments.' };
}
