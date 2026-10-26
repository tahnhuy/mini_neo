'use client';
import { useActionState } from 'react';
import { submitOrder } from '@/app/actions';
import type { Item } from '@/lib/data/types';
import { formatMoney } from '@/lib/formatMoney';

export function CreateForm({ items }: { items: Item[] }) {
  const [state, action, pending] = useActionState(submitOrder, { message: '' });
  return <form action={action} className="page-stack" aria-busy={pending}>
    {state.message && <p role="alert" className="alert alert-danger">{state.message}</p>}
    <div className="stock-table responsive-table panel" role="table" aria-label="Chọn mặt hàng">
      <div className="table-heading stock-grid" role="row"><span role="columnheader">Tên hàng</span><span className="align-right" role="columnheader">Đơn giá</span><span role="columnheader">Khả dụng</span><span role="columnheader">Số lượng</span></div>
      {items.map(item => <div className="stock-grid stock-row" key={item.sku} role="row">
        <div className="stock-product" role="cell"><strong>{item.name}</strong><small>{item.sku}</small><input type="hidden" name="sku" value={item.sku}/></div>
        <span className="stock-value stock-price money" role="cell"><span className="mobile-label muted">Đơn giá</span>{formatMoney(item.unitPrice)}</span>
        <span className="stock-value" role="cell"><span className="mobile-label muted">Khả dụng</span>{item.available}</span>
        <label className="stock-value" role="cell"><span className="mobile-label muted">Số lượng</span><input aria-label={`Số lượng ${item.sku}`} name={`qty:${item.sku}`} type="number" min="0" step="1" defaultValue="0" disabled={pending}/></label>
      </div>)}
    </div>
    <div className="create-actions"><p className="caption muted">Tổng tiền sẽ hiển thị sau khi tạo đơn</p><button className="button" disabled={pending}>{pending ? 'Đang tạo đơn…' : 'Tạo đơn'}</button></div>
  </form>;
}
