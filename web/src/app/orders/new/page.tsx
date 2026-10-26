import Link from 'next/link';
import { listStockItems } from '@/lib/data';
import { CreateForm } from '@/components/forms';
import { ErrorMessage, StateView } from '@/components/status';

export default async function NewOrder() {
  let data, error;
  try { data = await listStockItems(); } catch (caught) { error = caught; }
  return <section className="page-stack">
    <div className="page-intro"><Link className="back-link" href="/orders">← Đơn hàng</Link><h1>Tạo đơn</h1><p className="muted">Nhập số lượng mặt hàng cần đặt.</p></div>
    {data ? data.items.length ? <CreateForm items={data.items}/> : <StateView title="Chưa có mặt hàng" message="Chưa có mặt hàng nào trong kho của công ty."/> : <ErrorMessage error={error} title="Không thể tải mặt hàng"/>}
  </section>;
}
