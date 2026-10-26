import '../theme/tokens.css';

export const metadata = {
  title: 'NEO Mini · Đặt hàng và Giữ kho',
  description: 'Tạo đơn, duyệt đơn và theo dõi giữ hàng theo tenant.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <html lang="vi"><body>{children}</body></html>;
}
