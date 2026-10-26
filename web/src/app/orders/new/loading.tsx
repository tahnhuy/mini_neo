import Link from 'next/link';
import { LoadingView } from '@/components/status';
export default function Loading() {
  return <section className="page-stack">
    <div className="page-intro"><Link className="back-link" href="/orders">← Đơn hàng</Link><h1>Tạo đơn</h1><p className="muted">Nhập số lượng mặt hàng cần đặt.</p></div>
    <LoadingView/>
    <div className="create-actions"><p className="caption muted">Tổng tiền sẽ hiển thị sau khi tạo đơn</p><button className="button" disabled>Tạo đơn</button></div>
  </section>;
}
