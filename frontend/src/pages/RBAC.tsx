import { FiCheck, FiEdit2, FiPlus, FiTrash2, FiX } from 'solid-icons/fi';
import { For, Show, createResource, createSignal, type Component } from 'solid-js';
import Layout from '../components/Layout';
import { api, type RoleWithPermissions, type User } from '../lib/api';

const RBAC: Component = () => {
  const [activeTab, setActiveTab] = createSignal<'roles' | 'users'>('roles');
  const [roles, { refetch: refetchRoles }] = createResource(async () => {
    const response = await api.listRoles();
    return response.data || [];
  });
  const [users, { refetch: refetchUsers }] = createResource(async () => {
    const response = await api.listUsers();
    return response.data || [];
  });
  const [allPermissions] = createResource(async () => {
    const response = await api.getAllPermissions();
    return response.data || [];
  });

  const [showRoleModal, setShowRoleModal] = createSignal(false);
  const [editingRole, setEditingRole] = createSignal<RoleWithPermissions | null>(null);
  const [roleName, setRoleName] = createSignal('');
  const [roleDescription, setRoleDescription] = createSignal('');
  const [selectedPermissions, setSelectedPermissions] = createSignal<string[]>([]);
  const [roleError, setRoleError] = createSignal('');

  const [showUserRoleModal, setShowUserRoleModal] = createSignal(false);
  const [selectedUser, setSelectedUser] = createSignal<User | null>(null);

  const openCreateRoleModal = () => {
    setEditingRole(null);
    setRoleName('');
    setRoleDescription('');
    setSelectedPermissions([]);
    setRoleError('');
    setShowRoleModal(true);
  };

  const openEditRoleModal = (role: RoleWithPermissions) => {
    setEditingRole(role);
    setRoleName(role.name);
    setRoleDescription(role.description);
    setSelectedPermissions(role.permissions || []);
    setRoleError('');
    setShowRoleModal(true);
  };

  const handleSaveRole = async () => {
    if (!roleName().trim()) {
      setRoleError('Role name is required');
      return;
    }

    try {
      const existing = editingRole();
      if (existing) {
        await api.updateRole(existing.id, roleName(), roleDescription());
        await api.setRolePermissions(existing.id, selectedPermissions());
      } else {
        await api.createRole(roleName(), roleDescription(), selectedPermissions());
      }
      setShowRoleModal(false);
      refetchRoles();
    } catch (e) {
      setRoleError(e instanceof Error ? e.message : 'Failed to save role');
    }
  };

  const handleDeleteRole = async (id: number) => {
    if (!confirm('Are you sure you want to delete this role?')) return;

    try {
      await api.deleteRole(id);
      refetchRoles();
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to delete role');
    }
  };

  const togglePermission = (permission: string) => {
    const current = selectedPermissions();
    if (current.includes(permission)) {
      setSelectedPermissions(current.filter(p => p !== permission));
    } else {
      setSelectedPermissions([...current, permission]);
    }
  };

  const openUserRoleModal = (user: User) => {
    setSelectedUser(user);
    setShowUserRoleModal(true);
  };

  const handleAssignRole = async (roleId: number) => {
    const user = selectedUser();
    if (!user) return;

    try {
      await api.assignRole(user.id, roleId);
      refetchUsers();
      setShowUserRoleModal(false);
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to assign role');
    }
  };

  const handleRemoveRole = async (userId: number, roleId: number) => {
    if (!confirm('Remove this role from the user?')) return;

    try {
      await api.removeRole(userId, roleId);
      refetchUsers();
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to remove role');
    }
  };

  return (
    <Layout>
      <div class="rbac-page">
        <header class="page-header">
          <h1>RBAC Management</h1>
          <p>Manage roles, permissions, and user access</p>
        </header>

        <div class="tabs">
          <button
            class={`tab ${activeTab() === 'roles' ? 'active' : ''}`}
            onClick={() => setActiveTab('roles')}
          >
            Roles
          </button>
          <button
            class={`tab ${activeTab() === 'users' ? 'active' : ''}`}
            onClick={() => setActiveTab('users')}
          >
            Users
          </button>
        </div>

        <Show when={activeTab() === 'roles'}>
          <div class="section">
            <div class="section-header">
              <h2>Roles</h2>
              <button class="btn-primary" onClick={openCreateRoleModal}>
                <FiPlus /> Add
              </button>
            </div>

            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Description</th>
                    <th>Permissions</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  <Show when={!roles.loading} fallback={<tr><td colspan="4" class="loading">Loading...</td></tr>}>
                    <For each={roles()} fallback={<tr><td colspan="4" class="empty">No roles found</td></tr>}>
                      {(role) => (
                        <tr>
                          <td class="name">{role.name}</td>
                          <td class="desc">{role.description || '—'}</td>
                          <td>
                            <div class="tags">
                              <For each={role.permissions || []}>
                                {(permission) => <span class="tag">{permission}</span>}
                              </For>
                              <Show when={!role.permissions?.length}>
                                <span class="muted">None</span>
                              </Show>
                            </div>
                          </td>
                          <td class="actions">
                            <button class="btn-icon" onClick={() => openEditRoleModal(role)} title="Edit">
                              <FiEdit2 />
                            </button>
                            <button class="btn-icon danger" onClick={() => handleDeleteRole(role.id)} title="Delete">
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
        </Show>

        <Show when={activeTab() === 'users'}>
          <div class="section">
            <div class="section-header">
              <h2>Users</h2>
            </div>

            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Username</th>
                    <th>Email</th>
                    <th>Status</th>
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
                          <td class="desc">{user.email}</td>
                          <td>
                            <span class={`status ${user.is_active ? 'active' : ''}`}>
                              {user.is_active ? 'Active' : 'Inactive'}
                            </span>
                          </td>
                          <td>
                            <div class="tags">
                              <For each={user.roles || []}>
                                {(role) => (
                                  <span class="tag green">
                                    {role.name}
                                    <button class="tag-remove" onClick={() => handleRemoveRole(user.id, role.id)}>
                                      <FiX />
                                    </button>
                                  </span>
                                )}
                              </For>
                              <Show when={!user.roles?.length}>
                                <span class="muted">None</span>
                              </Show>
                            </div>
                          </td>
                          <td class="actions">
                            <button class="btn-icon" onClick={() => openUserRoleModal(user)} title="Assign Role">
                              <FiPlus />
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
        </Show>

        {/* Role Modal */}
        <Show when={showRoleModal()}>
          <div class="modal-overlay" onClick={() => setShowRoleModal(false)}>
            <div class="modal" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>{editingRole() ? 'Edit Role' : 'Create Role'}</h3>
                <button class="btn-icon" onClick={() => setShowRoleModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <Show when={roleError()}>
                  <div class="alert error">{roleError()}</div>
                </Show>
                <div class="form-field">
                  <label>Name</label>
                  <input
                    type="text"
                    value={roleName()}
                    onInput={(e) => setRoleName(e.currentTarget.value)}
                    placeholder="Enter role name"
                  />
                </div>
                <div class="form-field">
                  <label>Description</label>
                  <textarea
                    value={roleDescription()}
                    onInput={(e) => setRoleDescription(e.currentTarget.value)}
                    placeholder="Enter description"
                    rows={2}
                  />
                </div>
                <div class="form-field">
                  <label>Permissions</label>
                  <div class="checkbox-list">
                    <For each={allPermissions()}>
                      {(permission) => (
                        <label class="checkbox-item">
                          <input
                            type="checkbox"
                            checked={selectedPermissions().includes(permission)}
                            onChange={() => togglePermission(permission)}
                          />
                          <span>{permission}</span>
                        </label>
                      )}
                    </For>
                  </div>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowRoleModal(false)}>Cancel</button>
                <button class="btn-primary" onClick={handleSaveRole}>
                  <FiCheck /> {editingRole() ? 'Update' : 'Create'}
                </button>
              </div>
            </div>
          </div>
        </Show>

        {/* User Role Modal */}
        <Show when={showUserRoleModal()}>
          <div class="modal-overlay" onClick={() => setShowUserRoleModal(false)}>
            <div class="modal" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>Assign Role to {selectedUser()?.username}</h3>
                <button class="btn-icon" onClick={() => setShowUserRoleModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <div class="role-options">
                  <For each={roles()}>
                    {(role) => {
                      const hasRole = selectedUser()?.roles?.some(r => r.id === role.id);
                      return (
                        <button
                          class={`role-option ${hasRole ? 'assigned' : ''}`}
                          onClick={() => !hasRole && handleAssignRole(role.id)}
                          disabled={hasRole}
                        >
                          <div class="role-info">
                            <span class="role-name">{role.name}</span>
                            <span class="role-desc">{role.description}</span>
                          </div>
                          <Show when={hasRole}>
                            <FiCheck class="check-icon" />
                          </Show>
                        </button>
                      );
                    }}
                  </For>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowUserRoleModal(false)}>Close</button>
              </div>
            </div>
          </div>
        </Show>
      </div>

      <style>{`
        .rbac-page {
          max-width: 1000px;
        }

        .page-header {
          margin-bottom: 24px;
        }

        .page-header h1 {
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 6px;
        }

        .page-header p {
          color: var(--text-secondary);
          font-size: 14px;
        }

        .tabs {
          display: flex;
          gap: 4px;
          margin-bottom: 20px;
        }

        .tab {
          padding: 8px 16px;
          border: none;
          border-radius: var(--radius);
          background: transparent;
          color: var(--text-muted);
          font-size: 13px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.15s;
        }

        .tab:hover {
          color: var(--text);
          background: var(--surface);
        }

        .tab.active {
          color: white;
          background: var(--primary);
        }

        .section {
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
        }

        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 16px 20px;
          border-bottom: 1px solid var(--border);
        }

        .section-header h2 {
          font-size: 14px;
          font-weight: 500;
        }

        .btn-primary {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 6px 12px;
          border: none;
          border-radius: var(--radius);
          background: var(--primary);
          color: white;
          font-size: 13px;
          font-weight: 500;
          transition: background-color 0.15s;
        }

        .btn-primary:hover {
          background: var(--primary-hover);
        }

        .btn-secondary {
          padding: 8px 14px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: transparent;
          color: var(--text);
          font-size: 13px;
          transition: all 0.15s;
        }

        .btn-secondary:hover {
          background: var(--surface-hover);
        }

        .table-wrap {
          overflow-x: auto;
        }

        table {
          width: 100%;
          border-collapse: collapse;
        }

        th, td {
          padding: 12px 20px;
          text-align: left;
        }

        th {
          font-size: 11px;
          font-weight: 500;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 0.5px;
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

        td.desc {
          color: var(--text-secondary);
        }

        td.loading, td.empty {
          text-align: center;
          color: var(--text-muted);
          padding: 32px;
        }

        .tags {
          display: flex;
          flex-wrap: wrap;
          gap: 4px;
        }

        .tag {
          display: inline-flex;
          align-items: center;
          gap: 4px;
          padding: 3px 8px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
          background: var(--primary-light);
          color: var(--primary);
        }

        .tag.green {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .tag-remove {
          display: flex;
          padding: 0;
          border: none;
          background: transparent;
          color: inherit;
          cursor: pointer;
          opacity: 0.6;
        }

        .tag-remove:hover {
          opacity: 1;
        }

        .muted {
          color: var(--text-muted);
          font-size: 12px;
        }

        .status {
          display: inline-block;
          padding: 3px 8px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .status.active {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        td.actions {
          width: 1%;
          white-space: nowrap;
        }

        .btn-icon {
          width: 28px;
          height: 28px;
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
          max-width: 420px;
          max-height: 90vh;
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
          display: flex;
          flex-direction: column;
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
          font-weight: 500;
        }

        .modal-body {
          padding: 20px;
          overflow-y: auto;
        }

        .modal-footer {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
          padding: 16px 20px;
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
          margin-bottom: 16px;
        }

        .form-field:last-child {
          margin-bottom: 0;
        }

        .form-field label {
          display: block;
          font-size: 13px;
          font-weight: 500;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }

        .form-field input,
        .form-field textarea {
          width: 100%;
          padding: 10px 12px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: var(--background);
          color: var(--text);
          font-size: 13px;
          transition: border-color 0.15s;
        }

        .form-field input:focus,
        .form-field textarea:focus {
          outline: none;
          border-color: var(--primary);
        }

        .checkbox-list {
          max-height: 160px;
          overflow-y: auto;
          padding: 8px;
          background: var(--background);
          border-radius: var(--radius);
        }

        .checkbox-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 6px 8px;
          border-radius: var(--radius);
          cursor: pointer;
          font-size: 13px;
        }

        .checkbox-item:hover {
          background: var(--surface-hover);
        }

        .checkbox-item input {
          width: auto;
        }

        .role-options {
          display: flex;
          flex-direction: column;
          gap: 6px;
        }

        .role-option {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 12px 14px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: transparent;
          cursor: pointer;
          transition: all 0.15s;
          text-align: left;
          color: var(--text);
        }

        .role-option:hover:not(:disabled) {
          border-color: var(--primary);
        }

        .role-option:disabled {
          cursor: default;
          opacity: 0.6;
        }

        .role-option.assigned {
          border-color: var(--success);
          background: rgba(34, 197, 94, 0.05);
        }

        .role-info {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }

        .role-name {
          font-size: 13px;
          font-weight: 500;
        }

        .role-desc {
          font-size: 12px;
          color: var(--text-muted);
        }

        .check-icon {
          color: var(--success);
        }
      `}</style>
    </Layout>
  );
};

export default RBAC;
