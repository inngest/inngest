import { describe, expect, it } from 'vitest';

import { isSafeCTAURL } from './safeUrl';

describe('isSafeCTAURL', () => {
  it('accepts http and https absolute URLs', () => {
    expect(isSafeCTAURL('http://example.com')).toBe(true);
    expect(isSafeCTAURL('https://inngest.com/docs/upgrade')).toBe(true);
    expect(isSafeCTAURL('https://foo.bar/path?q=1#frag')).toBe(true);
  });

  it('accepts mailto:', () => {
    expect(isSafeCTAURL('mailto:support@inngest.com')).toBe(true);
  });

  it('accepts site-relative paths', () => {
    expect(isSafeCTAURL('/docs/upgrade')).toBe(true);
    expect(isSafeCTAURL('/')).toBe(true);
  });

  it('rejects protocol-relative URLs that could escape the origin', () => {
    expect(isSafeCTAURL('//evil.example/path')).toBe(false);
  });

  it('rejects javascript: in any casing', () => {
    expect(isSafeCTAURL('javascript:alert(1)')).toBe(false);
    expect(isSafeCTAURL('JavaScript:alert(1)')).toBe(false);
    expect(isSafeCTAURL('  javascript:alert(1)')).toBe(false);
  });

  it('rejects data:, vbscript:, file:, and other schemes', () => {
    expect(isSafeCTAURL('data:text/html,<script>alert(1)</script>')).toBe(
      false,
    );
    expect(isSafeCTAURL('vbscript:msgbox(1)')).toBe(false);
    expect(isSafeCTAURL('file:///etc/passwd')).toBe(false);
    expect(isSafeCTAURL('ftp://example.com')).toBe(false);
    expect(isSafeCTAURL('tel:+15551234567')).toBe(false);
  });

  it('rejects malformed URLs', () => {
    expect(isSafeCTAURL('')).toBe(false);
    expect(isSafeCTAURL('http://')).toBe(false);
    expect(isSafeCTAURL('not a url at all')).toBe(false);
  });
});
