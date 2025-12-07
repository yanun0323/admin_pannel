import { type Component, Show, For } from 'solid-js';
import { A } from '@solidjs/router';
import { FiTrendingUp, FiShield, FiKey, FiArrowRight } from 'solid-icons/fi';
import Layout from '../components/Layout';
import { authStore } from '../stores/auth';

const Dashboard: Component = () => {
  const user = authStore.user;

  return (
    <Layout>
      <div class="dashboard">
        <header class="page-header">
          <h1>Welcome, {user()?.username}</h1>
          <p>Here's an overview of your account</p>
        </header>

        <div class="quick-stats">
          <div class="stat-item">
            <span class="stat-value">{user()?.roles?.length || 0}</span>
            <span class="stat-label">Roles</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{user()?.permissions?.length || 0}</span>
            <span class="stat-label">Permissions</span>
          </div>
        </div>

        <section class="section">
          <h2>Quick Access</h2>
          <div class="cards">
            <Show when={authStore.hasPermission('view:kline')}>
              <A href="/kline" class="card">
                <div class="card-icon blue">
                  <FiTrendingUp />
                </div>
                <div class="card-content">
                  <h3>K-Line Charts</h3>
                  <p>View real-time trading charts</p>
                </div>
                <FiArrowRight class="card-arrow" />
              </A>
            </Show>

            <Show when={authStore.hasPermission('manage:roles')}>
              <A href="/rbac" class="card">
                <div class="card-icon green">
                  <FiShield />
                </div>
                <div class="card-content">
                  <h3>RBAC Management</h3>
                  <p>Manage roles and permissions</p>
                </div>
                <FiArrowRight class="card-arrow" />
              </A>
            </Show>

            <A href="/change-password" class="card">
              <div class="card-icon orange">
                <FiKey />
              </div>
              <div class="card-content">
                <h3>Security</h3>
                <p>Update your password</p>
              </div>
              <FiArrowRight class="card-arrow" />
            </A>
          </div>
        </section>

        <div class="info-grid">
          <section class="section">
            <h2>Your Roles</h2>
            <div class="tag-list">
              <Show when={user()?.roles?.length} fallback={<span class="empty">No roles assigned</span>}>
                <For each={user()?.roles}>
                  {(role) => <span class="tag green">{role.name}</span>}
                </For>
              </Show>
            </div>
          </section>

          <section class="section">
            <h2>Your Permissions</h2>
            <div class="tag-list">
              <Show when={user()?.permissions?.length} fallback={<span class="empty">No permissions</span>}>
                <For each={user()?.permissions}>
                  {(permission) => <span class="tag blue">{permission}</span>}
                </For>
              </Show>
            </div>
          </section>
        </div>
      </div>

      <style>{`
        .dashboard {
          max-width: 900px;
        }

        .page-header {
          margin-bottom: 32px;
        }

        .page-header h1 {
          font-size: 28px;
          font-weight: 600;
          margin-bottom: 6px;
        }

        .page-header p {
          color: var(--text-secondary);
          font-size: 14px;
        }

        .quick-stats {
          display: flex;
          gap: 32px;
          margin-bottom: 40px;
        }

        .stat-item {
          display: flex;
          flex-direction: column;
        }

        .stat-value {
          font-size: 32px;
          font-weight: 600;
          line-height: 1;
        }

        .stat-label {
          font-size: 13px;
          color: var(--text-muted);
          margin-top: 4px;
        }

        .section {
          margin-bottom: 32px;
        }

        .section h2 {
          font-size: 13px;
          font-weight: 500;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 12px;
        }

        .cards {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .card {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 14px 16px;
          background: var(--surface);
          border-radius: var(--radius);
          transition: background-color 0.15s;
          text-decoration: none;
          color: var(--text);
        }

        .card:hover {
          background: var(--surface-hover);
        }

        .card-icon {
          width: 36px;
          height: 36px;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: var(--radius);
          font-size: 16px;
          flex-shrink: 0;
        }

        .card-icon.blue {
          background: rgba(59, 130, 246, 0.1);
          color: var(--primary);
        }

        .card-icon.green {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .card-icon.orange {
          background: rgba(245, 158, 11, 0.1);
          color: var(--warning);
        }

        .card-content {
          flex: 1;
          min-width: 0;
        }

        .card-content h3 {
          font-size: 14px;
          font-weight: 500;
          margin-bottom: 2px;
        }

        .card-content p {
          font-size: 13px;
          color: var(--text-muted);
        }

        .card-arrow {
          color: var(--text-muted);
          flex-shrink: 0;
        }

        .info-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 24px;
        }

        .tag-list {
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
        }

        .tag {
          padding: 4px 10px;
          border-radius: 999px;
          font-size: 12px;
          font-weight: 500;
        }

        .tag.blue {
          background: rgba(59, 130, 246, 0.1);
          color: var(--primary);
        }

        .tag.green {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .empty {
          font-size: 13px;
          color: var(--text-muted);
        }

        @media (max-width: 640px) {
          .quick-stats {
            gap: 24px;
          }

          .stat-value {
            font-size: 24px;
          }
        }
      `}</style>
    </Layout>
  );
};

export default Dashboard;
