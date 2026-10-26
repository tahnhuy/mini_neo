import Link from 'next/link';
import type { Me } from '@/lib/data/types';

export function companyName(tenantId: string) {
  const names: Record<string, string> = {
    '11111111-1111-1111-1111-111111111111': 'Công ty A',
    '22222222-2222-2222-2222-222222222222': 'Công ty B',
  };
  return names[tenantId] ?? tenantId;
}

export function CapabilityTags({ capabilities }: { capabilities: string[] }) {
  const labels: Record<string, string> = {
    'order.create': 'Tạo đơn',
    'order.approve': 'Duyệt đơn',
  };
  return <div className="capability-tags">{capabilities.map(capability => (
    <span className="capability" key={capability}>{labels[capability] ?? capability}</span>
  ))}</div>;
}

export function AppHeader({ me, account = false }: { me?: Me; account?: boolean }) {
  return <header className={`app-header${account ? ' app-header-account' : ''}`}>
    <div className="header-inner">
      <Link className="brand" href="/">NEO Mini</Link>
      {me && <div className="header-identity">
        <div className="identity-name"><strong>{me.userId}</strong><span className="muted"><span className="mobile-separator">· </span>{companyName(me.tenantId)}</span></div>
        <CapabilityTags capabilities={me.capabilities}/>
      </div>}
      {account && <Link className="button button-secondary switch-user" href="/">Đổi người dùng</Link>}
    </div>
  </header>;
}
