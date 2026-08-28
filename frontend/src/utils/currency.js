export function formatCurrencyFromCents(valueCents, currencyCode = 'USD') {
  const numericCents = Number(valueCents);
  const safeCents = Number.isSafeInteger(numericCents) ? numericCents : 0;
  const value = safeCents / 100;

  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currencyCode,
    }).format(value);
  } catch {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  }
}

export function centsToDecimalString(valueCents) {
  const numericCents = Number(valueCents);
  if (!Number.isSafeInteger(numericCents)) return '';

  const sign = numericCents < 0 ? '-' : '';
  const absoluteCents = Math.abs(numericCents);
  const wholeUnits = Math.floor(absoluteCents / 100);
  const cents = String(absoluteCents % 100).padStart(2, '0');
  return `${sign}${wholeUnits}.${cents}`;
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
