import { describe, expect, it } from 'vitest';

import { signReceipt, verifyReceipt } from './chatReceipt';

describe('chat receipts', () => {
  it('verifies for the user the event was created for', () => {
    const receipt = signReceipt('evt_1', 'user_a');

    expect(verifyReceipt('evt_1', 'user_a', receipt)).toBe(true);
  });

  it('does not verify for another user', () => {
    const receipt = signReceipt('evt_1', 'user_a');

    expect(verifyReceipt('evt_1', 'user_b', receipt)).toBe(false);
  });

  it('does not verify for another event', () => {
    const receipt = signReceipt('evt_1', 'user_a');

    expect(verifyReceipt('evt_2', 'user_a', receipt)).toBe(false);
  });

  it('rejects a receipt of the wrong length without throwing', () => {
    expect(verifyReceipt('evt_1', 'user_a', 'nope')).toBe(false);
    expect(verifyReceipt('evt_1', 'user_a', '')).toBe(false);
  });
});
