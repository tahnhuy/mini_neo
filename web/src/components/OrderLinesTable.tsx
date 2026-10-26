import type { Order } from '@/lib/data/types';
import { formatMoney } from '@/lib/formatMoney';

export function OrderLinesTable({ order }: { order: Order }) {
  return <div className="lines-table responsive-table panel" role="table" aria-label="Các dòng đơn hàng">
    <div className="table-heading line-grid" role="row"><span role="columnheader">Mã hàng</span><span className="align-right" role="columnheader">Số lượng</span><span className="align-right" role="columnheader">Đơn giá</span><span className="align-right" role="columnheader">Thành tiền</span></div>
    {order.lines.map(line => <div className="line-grid line-row" role="row" key={line.sku}>
      <strong className="line-product" role="cell">{line.sku}</strong>
      <span className="line-value" role="cell"><span className="mobile-label muted">Số lượng</span>{line.quantity}</span>
      <span className="line-value money" role="cell"><span className="mobile-label muted">Đơn giá</span>{formatMoney(line.unitPrice)}</span>
      <span className="line-value money" role="cell"><span className="mobile-label muted">Thành tiền</span>{formatMoney(line.lineAmount)}</span>
    </div>)}
    <div className="order-total" role="row"><strong role="cell">Tổng tiền</strong><span role="cell" className="money">{formatMoney(order.totalAmount)}</span></div>
  </div>;
}
