import Link from 'next/link';
import { RefreshButton } from './RefreshButton';

const statuses: Record<string, { label: string; tone: string }> = {
  PENDING_APPROVAL: { label: 'Chờ duyệt', tone: 'pending' },
  APPROVED: { label: 'Đã duyệt', tone: 'approved' },
  RESERVED: { label: 'Đã giữ hàng', tone: 'reserved' },
  OUT_OF_STOCK: { label: 'Thiếu hàng', tone: 'out-of-stock' },
  REJECTED: { label: 'Đã từ chối', tone: 'rejected' },
};

export function Status({ status }: { status: string }) {
  const item = statuses[status];
  if (!item) return <span className="badge">{status}</span>;
  return <span className={`badge badge-${item.tone}`}>
    <img src={`/status/${item.tone}.svg`} width="6" height="6" alt=""/>{item.label}
  </span>;
}

export function StateView({ title, message, children, error = false }: {
  title: string; message?: string; children?: React.ReactNode; error?: boolean;
}) {
  return <div className="state-view panel" role={error ? 'alert' : undefined}>
    <h2>{title}</h2>{message && <p className="muted">{message}</p>}{children}
  </div>;
}

export function ErrorMessage({ error, title = 'Không thể tải dữ liệu' }: { error: unknown; title?: string }) {
  return <StateView title={title} message={error instanceof Error ? error.message : 'Vui lòng thử lại.'} error>
    <div className="state-actions"><Link className="button button-secondary" href="/">Đổi người dùng</Link><RefreshButton label="Thử lại"/></div>
  </StateView>;
}

export function LoadingView({ message = 'Đang tải dữ liệu…' }: { message?: string }) {
  return <div className="loading-view panel" role="status" aria-busy="true">
    <p className="muted">{message}</p>
    {[1, 2, 3].map(row => <div className="skeleton-row" key={row} aria-hidden="true"><div className="skeleton skeleton-title"/><div className="skeleton skeleton-description"/></div>)}
  </div>;
}
