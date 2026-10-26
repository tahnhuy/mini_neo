'use client';
import { useActionState, useState } from 'react';
import { decideOrder } from '@/app/actions';

export function OrderActions({ id, allowedActions }: { id: string; allowedActions: string[] }) {
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState('');
  const [approveState, approve, approving] = useActionState(decideOrder.bind(null, id, 'APPROVE'), { message: '' });
  const [rejectState, reject, pending] = useActionState(decideOrder.bind(null, id, 'REJECT'), { message: '' });
  const canApprove = allowedActions.includes('APPROVE');
  const canReject = allowedActions.includes('REJECT');
  // Array.from đếm điểm mã Unicode như rune Go, không chuẩn hóa chuỗi.
  const count = Array.from(reason).length;
  const invalid = count < 10 || count > 500;
  if (!canApprove && !canReject) return null;
  return <div className="page-stack">
    {approveState.message && <p className="alert alert-danger" role="alert">{approveState.message}</p>}
    <div className="decision-actions">
      {canReject && <button type="button" className="button button-danger" disabled={approving || pending} aria-expanded={rejecting} aria-controls="reject-form" onClick={() => setRejecting(true)}>Từ chối</button>}
      {canApprove && <form action={approve}><button className="button" disabled={approving || pending}>{approving ? 'Đang duyệt…' : 'Duyệt'}</button></form>}
    </div>
    {canReject && rejecting && <form id="reject-form" action={reject} className="panel reject-form" aria-busy={pending}>
      <div className="reject-heading"><label htmlFor="reason">Lý do từ chối</label><p className="caption muted" id="reason-help">Nhập từ 10 đến 500 ký tự.</p></div>
      <textarea id="reason" name="reason" autoFocus value={reason} disabled={pending} onChange={event => setReason(event.target.value)} aria-describedby="reason-help reason-count reason-error" aria-invalid={invalid && count > 0}/>
      <div className="reason-meta"><p className="caption" id="reason-error">{invalid && count > 0 ? 'Lý do phải có từ 10 đến 500 ký tự.' : ''}</p><span id="reason-count" className="caption muted">{count} / 500</span></div>
      {rejectState.message && <p className="alert alert-danger" role="alert">{rejectState.message}</p>}
      <div className="reject-actions"><button className="button button-danger" disabled={invalid || pending || approving}>{pending ? 'Đang từ chối…' : 'Xác nhận từ chối'}</button><button className="button button-secondary" type="button" disabled={pending} onClick={() => { setRejecting(false); setReason(''); }}>Hủy</button></div>
    </form>}
  </div>;
}
