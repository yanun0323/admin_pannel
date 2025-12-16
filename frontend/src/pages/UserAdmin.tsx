import { A } from '@solidjs/router';
import { FiCheck, FiEdit2, FiPlus, FiRefreshCw, FiTrash2, FiX } from 'solid-icons/fi';
import { For, Show, createResource, createSignal, type Component } from 'solid-js';
import Layout from '../components/Layout';
import { api, type TOTPSetup, type User } from '../lib/api';

const UserAdmin: Component = () => {
  const [users, { refetch: refetchUsers }] = createResource(async () => {
    const response = await api.listUsers();
    return response.data || [];
  });
  const [roles] = createResource(async () => {
    const response = await api.listRoles();
    return response.data || [];
  });

  const [showCreateModal, setShowCreateModal] = createSignal(false);
  const [showEditModal, setShowEditModal] = createSignal(false);
  const [showTOTPModal, setShowTOTPModal] = createSignal(false);

  const [editingUser, setEditingUser] = createSignal<User | null>(null);
  const [totpSetup, setTotpSetup] = createSignal<TOTPSetup | null>(null);

  const [username, setUsername] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [selectedRoles, setSelectedRoles] = createSignal<string[]>([]);
  const [isActive, setIsActive] = createSignal(true);
  const [formError, setFormError] = createSignal('');
  const [isSaving, setIsSaving] = createSignal(false);

  const resetForm = () => {
    setUsername('');
    setPassword('');
    setConfirmPassword('');
    setSelectedRoles([]);
    setIsActive(true);
    setFormError('');
    setEditingUser(null);
  };

  const openCreate = () => {
    resetForm();
    setShowCreateModal(true);
  };

  const openEdit = (user: User) => {
    resetForm();
    setEditingUser(user);
    setUsername(user.username);
    setIsActive(user.is_active);
    setSelectedRoles(user.roles?.map(r => r.id) || []);
    setShowEditModal(true);
  };

  const toggleRoleSelection = (id: string) => {
    const current = selectedRoles();
    if (current.includes(id)) {
      setSelectedRoles(current.filter(r => r !== id));
    } else {
      setSelectedRoles([...current, id]);
    }
  };

  const applyRolesDiff = async (userId: string, targetRoleIds: string[], currentRoleIds: string[]) => {
    const toAdd = targetRoleIds.filter(id => !currentRoleIds.includes(id));
    const toRemove = currentRoleIds.filter(id => !targetRoleIds.includes(id));

    for (const roleId of toAdd) {
      await api.assignRole(userId, roleId);
    }
    for (const roleId of toRemove) {
      await api.removeRole(userId, roleId);
    }
  };

  const handleCreate = async () => {
    setFormError('');
    if (!username().trim()) {
      setFormError('Username is required');
      return;
    }
    if (password() !== confirmPassword()) {
      setFormError('Passwords do not match');
      return;
    }

    setIsSaving(true);
    try {
      // Prefer dedicated admin endpoint; fall back to public register if backend lacks it.
      const response = await api.createUser({
        username: username().trim(),
        password: password(),
        roles: selectedRoles(),
        is_active: isActive(),
      });
      const created = response.data;
      if (created) {
        const roleIds = created.roles?.map(r => r.id) || [];
        await applyRolesDiff(created.id, selectedRoles(), roleIds);
      }
      setShowCreateModal(false);
      await refetchUsers();
    } catch (e: any) {
      setFormError(e?.message || 'Failed to create user. Backend may need user management endpoints.');
    } finally {
      setIsSaving(false);
    }
  };

  const handleUpdate = async () => {
    const user = editingUser();
    if (!user) return;

    setFormError('');
    setIsSaving(true);
    try {
      await api.updateUser(user.id, {
        username: username().trim(),
        is_active: isActive(),
      });
      await applyRolesDiff(user.id, selectedRoles(), user.roles?.map(r => r.id) || []);
      setShowEditModal(false);
      await refetchUsers();
    } catch (e: any) {
      setFormError(e?.message || 'Failed to update user. Backend may need user management endpoints.');
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('確定刪除這個使用者？此動作無法復原。')) return;
    try {
      await api.deleteUser(id);
      await refetchUsers();
    } catch (e: any) {
      alert(e?.message || 'Delete user failed. Backend may not expose DELETE /rbac/users/{id}.');
    }
  };

  const handleResetTOTP = async (userId: string) => {
    setIsSaving(true);
    try {
      const response = await api.resetUserTOTP(userId);
      setTotpSetup(response.data || null);
      setShowTOTPModal(true);
    } catch (e: any) {
      alert(e?.message || 'Reset 2FA failed. Backend may not expose this endpoint.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Layout>
      <div class="user-admin">
        <header class="page-header">
          <div>
            <h1>User Administration</h1>
            <p>Manage user lifecycle, roles, and 2FA status</p>
          </div>
          <div class="actions">
            <A href="/rbac" class="link">Go to RBAC</A>
            <button class="btn-primary" onClick={openCreate}>
              <FiPlus /> New User
            </button>
          </div>
        </header>

        <div class="section">
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Status</th>
                  <th>2FA</th>
                  <th>Roles</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <Show when={!users.loading} fallback={<tr><td colspan="5" class="loading">Loading...</td></tr>}>
                  <For each={users()} fallback={<tr><td colspan="5" class="empty">No users found</td></tr>}>
                    {(user) => (
                      <tr>
                        <td class="name">{user.username}</td>
                        <td>
                          <span class={`status ${user.is_active ? 'active' : ''}`}>
                            {user.is_active ? 'Active' : 'Inactive'}
                          </span>
                        </td>
                        <td>
                          <span class={`status ${user.totp_enabled ? 'active' : ''}`}>
                            {user.totp_enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td>
                          <div class="tags">
                            <For each={user.roles || []}>
                              {(role) => <span class="tag">{role.name}</span>}
                            </For>
                            <Show when={!user.roles?.length}>
                              <span class="muted">None</span>
                            </Show>
                          </div>
                        </td>
                        <td class="actions">
                          <button class="btn-icon" onClick={() => openEdit(user)} title="Edit">
                            <FiEdit2 />
                          </button>
                          <button class="btn-icon" onClick={() => handleResetTOTP(user.id)} title="Reset 2FA">
                            <FiRefreshCw />
                          </button>
                          <button class="btn-icon danger" onClick={() => handleDelete(user.id)} title="Delete">
                            <FiTrash2 />
                          </button>
                        </td>
                      </tr>
                    )}
                  </For>
                </Show>
              </tbody>
            </table>
          </div>
        </div>

        {/* Create Modal */}
        <Show when={showCreateModal()}>
          <div class="modal-overlay" onClick={() => setShowCreateModal(false)}>
            <div class="modal" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>Create User</h3>
                <button class="btn-icon" onClick={() => setShowCreateModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <Show when={formError()}>
                  <div class="alert error">{formError()}</div>
                </Show>
                <div class="form-field">
                  <label>Username</label>
                  <input
                    type="text"
                    value={username()}
                    onInput={(e) => setUsername(e.currentTarget.value)}
                    placeholder="Enter username"
                  />
                </div>
                <div class="form-field">
                  <label>Password</label>
                  <input
                    type="password"
                    value={password()}
                    onInput={(e) => setPassword(e.currentTarget.value)}
                    placeholder="Set initial password"
                  />
                </div>
                <div class="form-field">
                  <label>Confirm Password</label>
                  <input
                    type="password"
                    value={confirmPassword()}
                    onInput={(e) => setConfirmPassword(e.currentTarget.value)}
                    placeholder="Confirm password"
                  />
                </div>
                <div class="form-field inline">
                  <label class="checkbox">
                    <input type="checkbox" checked={isActive()} onChange={(e) => setIsActive(e.currentTarget.checked)} />
                    <span>Active immediately</span>
                  </label>
                </div>
                <div class="form-field">
                  <label>Assign Roles</label>
                  <div class="role-list">
                    <For each={roles()}>
                      {(role) => (
                        <label class="checkbox-item">
                          <input
                            type="checkbox"
                            checked={selectedRoles().includes(role.id)}
                            onChange={() => toggleRoleSelection(role.id)}
                          />
                          <div>
                            <div class="role-name">{role.name}</div>
                            <div class="role-desc">{role.description}</div>
                          </div>
                        </label>
                      )}
                    </For>
                  </div>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
                <button class="btn-primary" onClick={handleCreate} disabled={isSaving()}>
                  <FiCheck /> Create
                </button>
              </div>
            </div>
          </div>
        </Show>

        {/* Edit Modal */}
        <Show when={showEditModal()}>
          <div class="modal-overlay" onClick={() => setShowEditModal(false)}>
            <div class="modal" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>Edit User</h3>
                <button class="btn-icon" onClick={() => setShowEditModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <Show when={formError()}>
                  <div class="alert error">{formError()}</div>
                </Show>
                <div class="form-field">
                  <label>Username</label>
                  <input
                    type="text"
                    value={username()}
                    onInput={(e) => setUsername(e.currentTarget.value)}
                  />
                </div>
                <div class="form-field inline">
                  <label class="checkbox">
                    <input type="checkbox" checked={isActive()} onChange={(e) => setIsActive(e.currentTarget.checked)} />
                    <span>Active</span>
                  </label>
                </div>
                <div class="form-field">
                  <label>Roles</label>
                  <div class="role-list">
                    <For each={roles()}>
                      {(role) => (
                        <label class="checkbox-item">
                          <input
                            type="checkbox"
                            checked={selectedRoles().includes(role.id)}
                            onChange={() => toggleRoleSelection(role.id)}
                          />
                          <div>
                            <div class="role-name">{role.name}</div>
                            <div class="role-desc">{role.description}</div>
                          </div>
                        </label>
                      )}
                    </For>
                  </div>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowEditModal(false)}>Cancel</button>
                <button class="btn-primary" onClick={handleUpdate} disabled={isSaving()}>
                  <FiCheck /> Save
                </button>
              </div>
            </div>
          </div>
        </Show>

        {/* TOTP Modal */}
        <Show when={showTOTPModal() && totpSetup()}>
          <div class="modal-overlay" onClick={() => setShowTOTPModal(false)}>
            <div class="modal small" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>New 2FA Secret</h3>
                <button class="btn-icon" onClick={() => setShowTOTPModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <p class="muted">Share this QR or secret with the user to rebind their authenticator.</p>
                <div class="qr-container">
                  <img src={totpSetup()!.qr_code} alt="QR Code" class="qr-code" />
                </div>
                <div class="secret-box">
                  <div class="label">Secret</div>
                  <code>{totpSetup()!.secret}</code>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowTOTPModal(false)}>Close</button>
              </div>
            </div>
          </div>
        </Show>
      </div>

      <style>{`
        .user-admin {
          max-width: 1080px;
        }

        .page-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 12px;
          margin-bottom: 20px;
        }

        .page-header h1 {
          font-size: 24px;
          font-weight: 600;
        }

        .page-header p {
          color: var(--text-secondary);
          font-size: 13px;
        }

        .actions {
          display: flex;
          gap: 10px;
          align-items: center;
        }

        .link {
          color: var(--primary);
          font-size: 13px;
          margin-right: 5px;
          text-decoration: none;
        }

        .section {
          background: var(--surface);
          border-radius: var(--radius-lg);
          border: 1px solid var(--border);
          overflow: hidden;
        }

        .table-wrap {
          overflow-x: auto;
        }

        table {
          width: 100%;
          border-collapse: collapse;
        }

        th, td {
          padding: 12px 16px;
          text-align: left;
        }

        th {
          font-size: 11px;
          font-weight: 600;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 0.4px;
          border-bottom: 1px solid var(--border);
        }

        td {
          font-size: 13px;
          border-bottom: 1px solid var(--border);
        }

        tr:last-child td {
          border-bottom: none;
        }

        td.name {
          font-weight: 500;
        }

        td.loading, td.empty {
          text-align: center;
          color: var(--text-muted);
          padding: 24px;
        }

        .status {
          display: inline-block;
          padding: 4px 8px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 600;
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .status.active {
          background: rgba(34, 197, 94, 0.12);
          color: var(--success);
        }

        .tags {
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
        }

        .tag {
          padding: 4px 8px;
          border-radius: 999px;
          background: var(--primary-light);
          color: var(--primary);
          font-size: 11px;
          font-weight: 600;
        }

        .muted {
          color: var(--text-muted);
          font-size: 12px;
        }

        td.actions {
          width: 1%;
          white-space: nowrap;
        }

        .btn-icon {
          width: 30px;
          height: 30px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
          border: none;
          border-radius: var(--radius);
          background: transparent;
          color: var(--text-muted);
          cursor: pointer;
          transition: all 0.15s;
        }

        .btn-icon:hover {
          color: var(--text);
          background: var(--surface-hover);
        }

        .btn-icon.danger:hover {
          color: var(--danger);
          background: rgba(239, 68, 68, 0.1);
        }

        .btn-primary {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 8px 12px;
          border: none;
          border-radius: var(--radius);
          background: var(--primary);
          color: white;
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          transition: background-color 0.15s;
        }

        .btn-primary:hover:not(:disabled) {
          background: var(--primary-hover);
        }

        .btn-primary:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .btn-secondary {
          padding: 8px 12px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: transparent;
          color: var(--text);
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .btn-secondary:hover {
          background: var(--surface-hover);
        }

        /* Modal */
        .modal-overlay {
          position: fixed;
          inset: 0;
          background: rgba(0, 0, 0, 0.6);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 100;
          padding: 20px;
        }

        .modal {
          width: 100%;
          max-width: 520px;
          max-height: 90vh;
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
          display: flex;
          flex-direction: column;
        }

        .modal.small {
          max-width: 420px;
        }

        .modal-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 16px 20px;
          border-bottom: 1px solid var(--border);
        }

        .modal-header h3 {
          font-size: 15px;
          font-weight: 600;
        }

        .modal-body {
          padding: 20px;
          overflow-y: auto;
        }

        .modal-footer {
          display: flex;
          justify-content: flex-end;
          gap: 10px;
          padding: 14px 20px;
          border-top: 1px solid var(--border);
        }

        .alert {
          padding: 10px 12px;
          border-radius: var(--radius);
          font-size: 13px;
          margin-bottom: 16px;
        }

        .alert.error {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .form-field {
          margin-bottom: 14px;
        }

        .form-field:last-child {
          margin-bottom: 0;
        }

        .form-field label {
          display: block;
          font-size: 13px;
          font-weight: 600;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }

        .form-field input {
          width: 100%;
          padding: 10px 12px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: var(--background);
          color: var(--text);
          font-size: 13px;
          transition: border-color 0.15s;
        }

        .form-field input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .form-field.inline {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .checkbox {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          cursor: pointer;
        }

        .role-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
          max-height: 200px;
          overflow-y: auto;
          padding: 8px;
          background: var(--background);
          border-radius: var(--radius);
          border: 1px solid var(--border);
        }

        .checkbox-item {
          display: flex;
          gap: 10px;
          padding: 8px;
          border-radius: var(--radius);
          cursor: pointer;
        }

        .checkbox-item:hover {
          background: var(--surface-hover);
        }

        .role-name {
          font-size: 13px;
          font-weight: 600;
        }

        .role-desc {
          font-size: 12px;
          color: var(--text-muted);
        }

        .qr-container {
          display: flex;
          justify-content: center;
          margin: 12px 0;
        }

        .qr-code {
          width: 180px;
          height: 180px;
          background: white;
          padding: 10px;
          border-radius: var(--radius);
        }

        .secret-box {
          padding: 12px;
          border-radius: var(--radius);
          background: var(--background);
          border: 1px solid var(--border);
          display: grid;
          gap: 6px;
        }

        .secret-box .label {
          font-size: 12px;
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.5px;
          font-weight: 600;
        }
      `}</style>
    </Layout>
  );
};

export default UserAdmin;
