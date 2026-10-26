import { AppHeader } from '@/components/AppHeader';
import { getMe } from '@/lib/data';

export default async function OrdersLayout({ children }: { children: React.ReactNode }) {
  const me = await getMe().catch(() => undefined);
  return <><AppHeader me={me} account/><main className="page-content">{children}</main></>;
}
