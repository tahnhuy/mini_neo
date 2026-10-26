'use client';
export default function ErrorPage({ reset }: { reset: () => void }) {
  return <div role="alert" className="state-view panel"><h2>Không thể hiển thị trang</h2><p className="muted">Vui lòng thử lại.</p><button className="button" onClick={reset}>Thử lại</button></div>;
}
