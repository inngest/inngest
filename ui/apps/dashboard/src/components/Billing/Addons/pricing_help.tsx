function unitDescriptionFromTitle(title: string) {
  let result = title.toLowerCase();
  if (result === 'users') {
    result = 'user';
  }
  return result;
}

export function addonPriceStr(
  title: string,
  currentValue: number | boolean,
  quantityPer: number,
  price: number
) {
  const priceDollars = (price / 100).toFixed(2);
  if (typeof currentValue === 'boolean') {
    return `$${priceDollars} per month`;
  }
  if (quantityPer === 1) {
    return `$${priceDollars} per ${unitDescriptionFromTitle(title)}/month`;
  }
  return `$${priceDollars} per ${quantityPer} ${title.toLowerCase()}/month`;
}

export function addonQtyCostString(
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
