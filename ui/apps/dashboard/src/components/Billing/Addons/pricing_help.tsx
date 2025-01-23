function unitDescriptionFromTitle(title: string) {
  let result = title.toLowerCase();
  if (result === 'users') {
    result = 'user';
  }
  return result;
}

export function addonPriceStr(
  title: string,
  entitlement: {
    currentValue?: number | boolean;
  },
  addon: {
    quantityPer: number;
    price: number;
  }
) {
  if (entitlement.currentValue === undefined || entitlement.currentValue === -1) {
    console.error('calling addonPriceStr is nonsensical if current entitlement is unlimited');
    return '';
  }
  const priceDollars = (addon.price / 100).toFixed(2);
  if (typeof entitlement.currentValue === 'boolean') {
    return `$${priceDollars} per month`;
  }
  if (addon.quantityPer === 1) {
    return `$${priceDollars} per ${unitDescriptionFromTitle(title)}/month`;
  }
  return `$${priceDollars} per ${addon.quantityPer} ${title.toLowerCase()}/month`;
}

export function addonQtyCostString(
  title: string,
  qty: number,
  addon: {
    quantityPer: number;
    price: number;
  }
) {
  const price = qty * addon.price;
  const priceDollars = (price / 100).toFixed(2);
  return `$${priceDollars} per month`;
}
