import { A, useLocation, useNavigate } from '@solidjs/router';
import { FiHome, FiKey, FiLogOut, FiSettings, FiUser } from 'solid-icons/fi';
import { type Component, type JSX, Show } from 'solid-js';
import { authStore } from '../stores/auth';

interface LayoutProps {
  children: JSX.Element;
}

const Layout: Component<LayoutProps> = (props) => {
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = async () => {
    await authStore.logout();
    navigate('/login');
  };

  const isActive = (path: string) => location.pathname === path;

  return (
    <div class="layout">
      <aside class="sidebar">
        <div class="sidebar-logo">Nova</div>

        <nav class="sidebar-nav">
          <A href="/" class={`nav-item ${isActive('/') ? 'active' : ''}`} end title="Dashboard">
            <FiHome />
          </A>

          <Show when={authStore.hasPermission('manage:api_keys')}>
            <A href="/api-keys" class={`nav-item ${isActive('/api-keys') ? 'active' : ''}`} title="API Keys">
              <FiKey />
            </A>
          </Show>

          <Show when={authStore.hasPermission('manage:roles')}>
            <A href="/rbac" class={`nav-item ${isActive('/rbac') ? 'active' : ''}`} title="RBAC">
              <FiUser />
            </A>
          </Show>
        </nav>

        <div class="sidebar-footer">
          <A href="/settings" class={`nav-item ${isActive('/settings') ? 'active' : ''}`} title="Account Settings">
            <FiSettings />
          </A>
          <button class="nav-item logout" onClick={handleLogout} title="Logout">
            <FiLogOut />
          </button>
        </div>
      </aside>

      <main class="main-content">
        {props.children}
      </main>

      <style>{`
        .layout {
          display: flex;
          min-height: 100vh;
        }

        .sidebar {
          width: 56px;
          display: flex;
          flex-direction: column;
          align-items: center;
          padding: 16px 0;
          background: var(--surface);
          border-right: 1px solid var(--border);
          position: sticky;
          top: 0;
          height: 100vh;
        }

        .sidebar-logo {
          width: 36px;
          height: 36px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: var(--primary);
          color: white;
          font-weight: 600;
          font-size: 12px;
          border-radius: var(--radius);
          margin-bottom: 24px;
        }

        .sidebar-nav {
          flex: 1;
          display: flex;
          flex-direction: column;
          gap: 4px;
        }

        .nav-item {
          width: 36px;
          height: 36px;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: var(--radius);
          color: var(--text-muted);
          transition: all 0.15s;
          border: none;
          background: transparent;
          cursor: pointer;
        }

        .nav-item:hover {
          color: var(--text);
          background: var(--surface-hover);
        }

        .nav-item.active {
          color: var(--primary);
          background: var(--primary-light);
        }

        .nav-item svg {
          width: 18px;
          height: 18px;
        }

        .sidebar-footer {
          margin-top: auto;
        }

        .nav-item.logout:hover {
          color: var(--danger);
          background: rgba(239, 68, 68, 0.1);
        }

        .main-content {
          flex: 1;
          padding: 32px 40px;
          min-height: 100vh;
          overflow-y: auto;
        }

        @media (max-width: 768px) {
          .main-content {
            padding: 24px 20px;
          }
        }
      `}</style>
    </div>
  );
};

export default Layout;