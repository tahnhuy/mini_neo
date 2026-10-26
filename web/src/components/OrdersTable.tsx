import Link from 'next/link';
import type { Order } from '@/lib/data/types';
import { formatMoney } from '@/lib/formatMoney';
import { Status } from './status';

export function OrdersTable({ orders }: { orders: Order[] }) {
  return <div className="orders-table responsive-table panel" role="table" aria-label="Danh sách đơn hàng">
    <div className="table-heading order-grid" role="row">
      <span role="columnheader">Mã đơn</span><span role="columnheader">Người lập</span>
      <span className="align-right" role="columnheader">Tổng tiền</span><span role="columnheader">Trạng thái</span><span aria-hidden="true"/>
    </div>
    {orders.map(order => <Link className="order-grid order-row" href={`/orders/${order.id}`} key={order.id} role="row" aria-label={`Xem đơn ${order.code}`}>
      <strong className="order-code" role="cell">{order.code}</strong>
      <span className="order-creator" role="cell"><span className="mobile-label muted">Người lập</span>{order.createdBy}</span>
      <span className="order-money money" role="cell"><span className="mobile-label muted">Tổng tiền</span>{formatMoney(order.totalAmount)}</span>
      <span className="order-status" role="cell"><Status status={order.status}/></span>
      <span className="order-arrow muted" aria-hidden="true">→</span>
    </Link>)}
  </div>;
}
