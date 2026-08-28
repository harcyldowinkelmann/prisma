export function formatCurrency(value, currencyCode = 'USD') {
  const numericValue = Number(value);
  const safeValue = Number.isFinite(numericValue) ? numericValue : 0;

  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currencyCode,
    }).format(safeValue);
  } catch {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(safeValue);
  }
}

export function getCurrencySymbol(currencyCode = 'USD') {
  try {
    const symbolPart = new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currencyCode,
      currencyDisplay: 'narrowSymbol',
    }).formatToParts(0).find(part => part.type === 'currency');

    return symbolPart?.value || currencyCode;
  } catch {
    return '$';
  }
}
