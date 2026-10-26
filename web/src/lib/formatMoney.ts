import type { Money } from './data/types';
export function formatMoney(money: Money): string {
  const [whole, fraction = '00'] = money.amount.split('.');
  const amount = BigInt(whole).toLocaleString('vi-VN');
  return `${amount}${fraction === '00' ? '' : `,${fraction}`} ${money.currencyCode === 'VND' ? '₫' : money.currencyCode}`;
}
