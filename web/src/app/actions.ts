'use server';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import { revalidatePath } from 'next/cache';
import { approveOrder, createOrder, rejectOrder } from '@/lib/data';
export type ActionState = { message: string };
export async function selectUser(form: FormData) {
  const user = String(form.get('user'));
  if (!['an', 'binh', 'chi', 'dung'].includes(user)) return;
  (await cookies()).set('dev_user', user, { httpOnly: true, sameSite: 'lax', path: '/' });
  redirect('/orders');
}
export async function submitOrder(_: ActionState, form: FormData): Promise<ActionState> {
  const lines = form.getAll('sku').map(sku => ({ sku: String(sku), quantity: Number(form.get(`qty:${sku}`)) })).filter(line => line.quantity !== 0);
  let id: string;
  try { id = (await createOrder(lines)).order.id; }
  catch (error) { return { message: error instanceof Error ? error.message : 'Không thể tạo đơn' }; }
  redirect(`/orders/${id}`);
}
export async function decideOrder(id: string, action: string, _: ActionState, form: FormData): Promise<ActionState> {
  try {
    if (action === 'APPROVE') await approveOrder(id);
    // Trình duyệt đổi xuống dòng LF của textarea thành CRLF khi gửi form; đổi lại LF để
    // service đếm đúng số ký tự người dùng thấy ở bộ đếm (không chuẩn hóa Unicode, Q17).
    else await rejectOrder(id, String(form.get('reason') ?? '').replace(/\r\n/g, '\n'));
  } catch (error) { return { message: error instanceof Error ? error.message : 'Không thể xử lý đơn' }; }
  revalidatePath(`/orders/${id}`);
  return { message: '' };
}
