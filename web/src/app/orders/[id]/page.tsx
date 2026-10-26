import Link from 'next/link';
import { ApiError, getOrder } from '@/lib/data';
import { OrderActions } from '@/components/OrderActions';
import { RefreshButton } from '@/components/RefreshButton';
import { OrderLinesTable } from '@/components/OrderLinesTable';
import { ErrorMessage, StateView, Status } from '@/components/status';

export default async function Detail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  let data;
  try { data = await getOrder(id); } catch (error) {
    return <section className="page-stack"><Link className="back-link" href="/orders">← Đơn hàng</Link>
      {error instanceof ApiError && error.code === 'NOT_FOUND'
        ? <StateView title="Không tìm thấy đơn hàng" message={error.message}><div className="state-actions"><Link className="button button-secondary" href="/orders">Đơn hàng</Link><RefreshButton label="Thử lại"/></div></StateView>
        : <ErrorMessage error={error} title="Không thể tải đơn hàng"/>}
    </section>;
  }
  const { order } = data;
  const notices: Record<string, string> = {
    APPROVED: 'Đơn đã duyệt. Kho đang xử lý giữ hàng.',
    RESERVED: 'Kho đã giữ hàng cho đơn này.',
    OUT_OF_STOCK: 'Kho không đủ hàng. Không mặt hàng nào được giữ.',
  };
  return <section className="page-stack">
    <Link className="back-link" href="/orders">← Đơn hàng</Link>
    <div className="order-intro"><div className="order-title"><h1>{order.code}</h1><Status status={order.status}/></div><p className="muted">Người lập: {order.createdBy}</p></div>
    {notices[order.status] && <div className="stock-notice"><div className="alert alert-info"><span aria-hidden="true">!</span><p>{notices[order.status]}</p></div>{order.status === 'APPROVED' && <RefreshButton/>}</div>}
    <OrderLinesTable order={order}/>
    <OrderActions key={order.status} id={id} allowedActions={order.allowedActions}/>
    {order.rejectReason && <div className="panel reject-reason"><h2>Lý do từ chối</h2><p>{order.rejectReason}</p></div>}
  </section>;
}
