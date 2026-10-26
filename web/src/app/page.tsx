import { selectUser } from './actions';
import { AppHeader, CapabilityTags } from '@/components/AppHeader';

const users = [
  { name: 'an', company: 'Công ty A', capabilities: ['order.create'] },
  { name: 'binh', company: 'Công ty A', capabilities: ['order.create', 'order.approve'] },
  { name: 'chi', company: 'Công ty A', capabilities: ['order.approve'] },
  { name: 'dung', company: 'Công ty B', capabilities: ['order.create', 'order.approve'] },
];

export default function Home() {
  return <><AppHeader/><main className="home-content">
    <div className="intro">
      <h1>Chọn người dùng mẫu</h1>
      <p className="muted">Chọn một tài khoản để tạo đơn, duyệt đơn và theo dõi giữ hàng theo quyền được cấp. Không cần đăng nhập.</p>
    </div>
    <form action={selectUser} className="user-grid">
      {users.map(user => <article className="user-card" key={user.name}>
        <div className="user-card-identity"><h2>{user.name}</h2><p className="muted">{user.company}</p></div>
        <CapabilityTags capabilities={user.capabilities}/>
        <button className="button button-secondary" name="user" value={user.name}>Chọn {user.name}</button>
      </article>)}
    </form>
    <p className="caption muted">Dữ liệu và quyền hạn được phân tách theo công ty.</p>
  </main></>;
}
