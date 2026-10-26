import 'server-only';
import { cache } from 'react';
import { cookies } from 'next/headers';
import type { Item, Me, Order } from './types';

export class ApiError extends Error {
  constructor(public code: string, message: string) { super(message); }
}
async function request<T>(path: string, body?: unknown): Promise<T> {
  const user = (await cookies()).get('dev_user')?.value ?? '';
  let response: Response;
  try {
    response = await fetch(`${process.env.BFF_URL ?? 'http://127.0.0.1:8080'}/api${path}`, {
      method: body === undefined ? 'GET' : 'POST', cache: 'no-store',
      headers: { 'X-Dev-User': user, 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(7000),
    });
  } catch { throw new ApiError('UNAVAILABLE', 'Không thể kết nối BFF. Vui lòng thử lại.'); }
  // Body có thể không phải JSON (proxy lỗi, hết giờ khi đang đọc); không để lộ lỗi parse ra giao diện.
  const data = await response.json().catch(() => undefined);
  if (!response.ok) {
    throw new ApiError(data?.code ?? 'UNKNOWN', data?.message ?? 'BFF trả lỗi không đọc được. Vui lòng thử lại.');
  }
  if (data === undefined) throw new ApiError('UNKNOWN', 'BFF trả dữ liệu không đọc được. Vui lòng thử lại.');
  return data as T;
}
// cache: layout và trang /orders cùng gọi getMe() trong một request; chỉ gọi BFF một lần.
export const getMe = cache(() => request<Me>('/me'));
export const listOrders = () => request<{ orders: Order[] }>('/orders');
export const getOrder = (id: string) => request<{ order: Order }>(`/orders/${encodeURIComponent(id)}`);
export const createOrder = (lines: { sku: string; quantity: number }[]) => request<{ order: Order }>('/orders', { lines });
export const approveOrder = (id: string) => request<{ order: Order }>(`/orders/${encodeURIComponent(id)}/approve`, {});
export const rejectOrder = (id: string, reason: string) => request<{ order: Order }>(`/orders/${encodeURIComponent(id)}/reject`, { reason });
export const listStockItems = () => request<{ items: Item[] }>('/stock-items');
