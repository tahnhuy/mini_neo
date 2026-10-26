import Link from 'next/link';
import { LoadingView } from '@/components/status';
export default function Loading() {
  return <section className="page-stack"><Link className="back-link" href="/orders">← Đơn hàng</Link><LoadingView message="Đang tải đơn hàng…"/></section>;
}
