import Link from 'next/link';
import { getMe, listOrders } from '@/lib/data';
import { ErrorMessage, StateView } from '@/components/status';
import { OrdersTable } from '@/components/OrdersTable';

export default async function Orders() {
  const me = await getMe().catch(() => undefined);
  let data;
  try { data = await listOrders(); } catch (error) {
    return <section className="page-stack"><PageTitle canCreate={me?.capabilities.includes('order.create')}/><ErrorMessage error={error} title="Không thể tải danh sách đơn"/></section>;
  }
  return <section className="page-stack">
    <PageTitle canCreate={me?.capabilities.includes('order.create')}/>
    {data.orders.length ? <OrdersTable orders={data.orders}/> : <StateView title="Chưa có đơn nào" message="Các đơn hàng của công ty sẽ xuất hiện tại đây."/>}
  </section>;
}

function PageTitle({ canCreate }: { canCreate?: boolean }) {
  return <div className="page-title"><h1>Đơn hàng</h1>{canCreate && <Link className="button" href="/orders/new">Tạo đơn</Link>}</div>;
}
