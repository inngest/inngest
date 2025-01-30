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
