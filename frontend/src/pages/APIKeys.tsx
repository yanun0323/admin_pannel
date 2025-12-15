import { FiCheck, FiEdit2, FiEye, FiEyeOff, FiKey, FiPlus, FiTrash2, FiX } from 'solid-icons/fi';
import { For, Show, createResource, createSignal, type Component } from 'solid-js';
import Layout from '../components/Layout';
import { api, type APIKeyResponse, type CreateAPIKeyRequest, type UpdateAPIKeyRequest } from '../lib/api';

const APIKeys: Component = () => {
  const [apiKeys, { refetch }] = createResource(async () => {
    const response = await api.listAPIKeys();
    return response.data || [];
  });

  const [platforms] = createResource(async () => {
    const response = await api.getAPIKeyPlatforms();
    return response.data || [];
  });

  // Modal states
  const [showModal, setShowModal] = createSignal(false);
  const [editingKey, setEditingKey] = createSignal<APIKeyResponse | null>(null);
  const [modalError, setModalError] = createSignal('');
  const [isSubmitting, setIsSubmitting] = createSignal(false);

  // Form states
  const [name, setName] = createSignal('');
  const [platform, setPlatform] = createSignal('');
  const [apiKey, setApiKey] = createSignal('');
  const [apiSecret, setApiSecret] = createSignal('');
  const [isTestnet, setIsTestnet] = createSignal(false);
  const [isActive, setIsActive] = createSignal(true);

  // Secret visibility
  const [showApiKey, setShowApiKey] = createSignal(false);
  const [showApiSecret, setShowApiSecret] = createSignal(false);

  const resetForm = () => {
    setName('');
    setPlatform('');
    setApiKey('');
    setApiSecret('');
    setIsTestnet(false);
    setIsActive(true);
    setShowApiKey(false);
    setShowApiSecret(false);
    setModalError('');
  };

  const openCreateModal = () => {
    setEditingKey(null);
    resetForm();
    // Set default platform if available
    const platformList = platforms();
    if (platformList && platformList.length > 0) {
      setPlatform(platformList[0]);
    }
    setShowModal(true);
  };

  const openEditModal = (key: APIKeyResponse) => {
    setEditingKey(key);
    setName(key.name);
    setPlatform(key.platform);
    setApiKey(''); // Don't pre-fill sensitive data
    setApiSecret('');
    setIsTestnet(key.is_testnet);
    setIsActive(key.is_active);
    setShowApiKey(false);
    setShowApiSecret(false);
    setModalError('');
    setShowModal(true);
  };

  const handleSubmit = async () => {
    if (!name().trim()) {
      setModalError('Name is required');
      return;
    }
    if (!platform()) {
      setModalError('Platform is required');
      return;
    }

    const existing = editingKey();

    if (!existing) {
      // Create mode - require api_key and api_secret
      if (!apiKey().trim()) {
        setModalError('API Key is required');
        return;
      }
      if (!apiSecret().trim()) {
        setModalError('API Secret is required');
        return;
      }
    }

    setIsSubmitting(true);
    setModalError('');

    try {
      if (existing) {
        // Update mode
        const updateReq: UpdateAPIKeyRequest = {
          name: name(),
          is_testnet: isTestnet(),
          is_active: isActive(),
        };
        // Only include api_key and api_secret if they are provided
        if (apiKey().trim()) {
          updateReq.api_key = apiKey();
        }
        if (apiSecret().trim()) {
          updateReq.api_secret = apiSecret();
        }
        await api.updateAPIKey(existing.id, updateReq);
      } else {
        // Create mode
        const createReq: CreateAPIKeyRequest = {
          name: name(),
          platform: platform(),
          api_key: apiKey(),
          api_secret: apiSecret(),
          is_testnet: isTestnet(),
        };
        await api.createAPIKey(createReq);
      }
      setShowModal(false);
      refetch();
    } catch (e) {
      setModalError(e instanceof Error ? e.message : 'Failed to save API key');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    try {
      await api.deleteAPIKey(id);
      refetch();
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to delete API key');
    }
  };

  const toggleActive = async (key: APIKeyResponse) => {
    try {
      await api.updateAPIKey(key.id, { is_active: !key.is_active });
      refetch();
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to update API key');
    }
  };

  const getPlatformDisplay = (platform: string) => {
    const displays: Record<string, string> = {
      binance: 'Binance',
      btcc: 'BTCC',
      okx: 'OKX',
      bybit: 'Bybit',
    };
    return displays[platform] || platform;
  };

  return (
    <Layout>
      <div class="apikeys-page">
        <header class="page-header">
          <div class="header-content">
            <h1>API Keys</h1>
            <p>Manage your exchange API credentials for trading and monitoring</p>
          </div>
          <button class="btn-primary" onClick={openCreateModal}>
            <FiPlus /> Add API Key
          </button>
        </header>

        <div class="section">
          <Show when={!apiKeys.loading} fallback={<div class="loading">Loading...</div>}>
            <Show
              when={apiKeys()?.length}
              fallback={
                <div class="empty-state">
                  <FiKey class="empty-icon" />
                  <h3>No API Keys</h3>
                  <p>Add your first API key to start monitoring and trading</p>
                  <button class="btn-primary" onClick={openCreateModal}>
                    <FiPlus /> Add API Key
                  </button>
                </div>
              }
            >
              <div class="keys-grid">
                <For each={apiKeys()}>
                  {(key) => (
                    <div class={`key-card ${!key.is_active ? 'inactive' : ''}`}>
                      <div class="card-header">
                        <div class="card-title">
                          <span class="key-name">{key.name}</span>
                          <div class="key-badges">
                            <span class={`badge platform ${key.platform}`}>
                              {getPlatformDisplay(key.platform)}
                            </span>
                            <Show when={key.is_testnet}>
                              <span class="badge testnet">Testnet</span>
                            </Show>
                            <span class={`badge status ${key.is_active ? 'active' : 'inactive'}`}>
                              {key.is_active ? 'Active' : 'Inactive'}
                            </span>
                          </div>
                        </div>
                        <div class="card-actions">
                          <button
                            class="btn-icon"
                            onClick={() => toggleActive(key)}
                            title={key.is_active ? 'Deactivate' : 'Activate'}
                          >
                            <Show when={key.is_active} fallback={<FiEye />}>
                              <FiEyeOff />
                            </Show>
                          </button>
                          <button class="btn-icon" onClick={() => openEditModal(key)} title="Edit">
                            <FiEdit2 />
                          </button>
                          <button
                            class="btn-icon danger"
                            onClick={() => handleDelete(key.id)}
                            title="Delete"
                          >
                            <FiTrash2 />
                          </button>
                        </div>
                      </div>
                      <div class="card-body">
                        <div class="key-info">
                          <span class="label">API Key</span>
                          <code class="value">{key.api_key_masked}</code>
                        </div>
                        <div class="key-meta">
                          <span class="meta-item">
                            Created: {new Date(key.created_at).toLocaleDateString()}
                          </span>
                          <span class="meta-item">
                            Updated: {new Date(key.updated_at).toLocaleDateString()}
                          </span>
                        </div>
                      </div>
                    </div>
                  )}
                </For>
              </div>
            </Show>
          </Show>
        </div>

        {/* Create/Edit Modal */}
        <Show when={showModal()}>
          <div class="modal-overlay" onClick={() => setShowModal(false)}>
            <div class="modal" onClick={(e) => e.stopPropagation()}>
              <div class="modal-header">
                <h3>{editingKey() ? 'Edit API Key' : 'Add API Key'}</h3>
                <button class="btn-icon" onClick={() => setShowModal(false)}>
                  <FiX />
                </button>
              </div>
              <div class="modal-body">
                <Show when={modalError()}>
                  <div class="alert error">{modalError()}</div>
                </Show>

                <div class="form-field">
                  <label>Name *</label>
                  <input
                    type="text"
                    value={name()}
                    onInput={(e) => setName(e.currentTarget.value)}
                    placeholder="e.g., Main Trading Account"
                  />
                </div>

                <div class="form-field">
                  <label>Platform *</label>
                  <select
                    value={platform()}
                    onChange={(e) => setPlatform(e.currentTarget.value)}
                    disabled={!!editingKey()}
                  >
                    <option value="">Select platform...</option>
                    <For each={platforms()}>
                      {(p) => <option value={p}>{getPlatformDisplay(p)}</option>}
                    </For>
                  </select>
                  <Show when={editingKey()}>
                    <span class="hint">Platform cannot be changed after creation</span>
                  </Show>
                </div>

                <div class="form-field">
                  <label>
                    API Key {editingKey() ? '(leave empty to keep current)' : '*'}
                  </label>
                  <div class="input-with-toggle">
                    <input
                      type={showApiKey() ? 'text' : 'password'}
                      value={apiKey()}
                      onInput={(e) => setApiKey(e.currentTarget.value)}
                      placeholder={editingKey() ? '••••••••' : 'Enter API key'}
                      autocomplete="off"
                    />
                    <button
                      type="button"
                      class="toggle-visibility"
                      onClick={() => setShowApiKey(!showApiKey())}
                    >
                      <Show when={showApiKey()} fallback={<FiEye />}>
                        <FiEyeOff />
                      </Show>
                    </button>
                  </div>
                </div>

                <div class="form-field">
                  <label>
                    API Secret {editingKey() ? '(leave empty to keep current)' : '*'}
                  </label>
                  <div class="input-with-toggle">
                    <input
                      type={showApiSecret() ? 'text' : 'password'}
                      value={apiSecret()}
                      onInput={(e) => setApiSecret(e.currentTarget.value)}
                      placeholder={editingKey() ? '••••••••' : 'Enter API secret'}
                      autocomplete="off"
                    />
                    <button
                      type="button"
                      class="toggle-visibility"
                      onClick={() => setShowApiSecret(!showApiSecret())}
                    >
                      <Show when={showApiSecret()} fallback={<FiEye />}>
                        <FiEyeOff />
                      </Show>
                    </button>
                  </div>
                </div>

                <div class="form-row">
                  <label class="checkbox-label">
                    <input
                      type="checkbox"
                      checked={isTestnet()}
                      onChange={(e) => setIsTestnet(e.currentTarget.checked)}
                    />
                    <span>Testnet</span>
                  </label>

                  <Show when={editingKey()}>
                    <label class="checkbox-label">
                      <input
                        type="checkbox"
                        checked={isActive()}
                        onChange={(e) => setIsActive(e.currentTarget.checked)}
                      />
                      <span>Active</span>
                    </label>
                  </Show>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn-secondary" onClick={() => setShowModal(false)}>
                  Cancel
                </button>
                <button class="btn-primary" onClick={handleSubmit} disabled={isSubmitting()}>
                  <FiCheck /> {isSubmitting() ? 'Saving...' : editingKey() ? 'Update' : 'Create'}
                </button>
              </div>
            </div>
          </div>
        </Show>
      </div>

      <style>{`
        .apikeys-page {
          max-width: 1000px;
        }

        .page-header {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          margin-bottom: 24px;
          gap: 16px;
        }

        .header-content h1 {
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 6px;
        }

        .header-content p {
          color: var(--text-secondary);
          font-size: 14px;
        }

        .section {
          background: var(--surface);
          border-radius: var(--radius-lg);
          padding: 20px;
        }

        .loading {
          text-align: center;
          color: var(--text-muted);
          padding: 40px;
        }

        .empty-state {
          text-align: center;
          padding: 60px 20px;
        }

        .empty-icon {
          width: 48px;
          height: 48px;
          color: var(--text-muted);
          margin-bottom: 16px;
        }

        .empty-state h3 {
          font-size: 16px;
          font-weight: 500;
          margin-bottom: 8px;
        }

        .empty-state p {
          color: var(--text-secondary);
          font-size: 14px;
          margin-bottom: 20px;
        }

        .keys-grid {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }

        .key-card {
          border: 1px solid var(--border);
          border-radius: var(--radius);
          overflow: hidden;
          transition: border-color 0.15s;
        }

        .key-card:hover {
          border-color: var(--primary);
        }

        .key-card.inactive {
          opacity: 0.6;
        }

        .card-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 14px 16px;
          background: var(--background);
          border-bottom: 1px solid var(--border);
        }

        .card-title {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .key-name {
          font-size: 14px;
          font-weight: 500;
        }

        .key-badges {
          display: flex;
          gap: 6px;
          flex-wrap: wrap;
        }

        .badge {
          display: inline-block;
          padding: 2px 8px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
        }

        .badge.platform {
          background: var(--primary-light);
          color: var(--primary);
        }

        .badge.platform.binance {
          background: rgba(243, 186, 47, 0.15);
          color: #f3ba2f;
        }

        .badge.platform.okx {
          background: rgba(0, 177, 93, 0.15);
          color: #00b15d;
        }

        .badge.platform.bybit {
          background: rgba(255, 160, 0, 0.15);
          color: #f7a600;
        }

        .badge.platform.btcc {
          background: rgba(42, 130, 228, 0.15);
          color: #2a82e4;
        }

        .badge.testnet {
          background: rgba(251, 191, 36, 0.15);
          color: #f59e0b;
        }

        .badge.status {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .badge.status.active {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .card-actions {
          display: flex;
          gap: 4px;
        }

        .card-body {
          padding: 14px 16px;
        }

        .key-info {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 12px;
        }

        .key-info .label {
          font-size: 12px;
          color: var(--text-muted);
        }

        .key-info .value {
          font-family: monospace;
          font-size: 13px;
          padding: 4px 8px;
          background: var(--background);
          border-radius: var(--radius);
        }

        .key-meta {
          display: flex;
          gap: 16px;
        }

        .meta-item {
          font-size: 12px;
          color: var(--text-muted);
        }

        /* Buttons */
        .btn-primary {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 8px 14px;
          border: none;
          border-radius: var(--radius);
          background: var(--primary);
          color: white;
          font-size: 13px;
          font-weight: 500;
          cursor: pointer;
          transition: background-color 0.15s;
        }

        .btn-primary:hover:not(:disabled) {
          background: var(--primary-hover);
        }

        .btn-primary:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .btn-secondary {
          padding: 8px 14px;
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
          max-width: 450px;
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
        .form-field select {
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
        .form-field select:focus {
          outline: none;
          border-color: var(--primary);
        }

        .form-field select:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .hint {
          display: block;
          margin-top: 4px;
          font-size: 11px;
          color: var(--text-muted);
        }

        .input-with-toggle {
          position: relative;
          display: flex;
        }

        .input-with-toggle input {
          padding-right: 40px;
        }

        .toggle-visibility {
          position: absolute;
          right: 8px;
          top: 50%;
          transform: translateY(-50%);
          width: 28px;
          height: 28px;
          display: flex;
          align-items: center;
          justify-content: center;
          border: none;
          border-radius: var(--radius);
          background: transparent;
          color: var(--text-muted);
          cursor: pointer;
        }

        .toggle-visibility:hover {
          color: var(--text);
        }

        .form-row {
          display: flex;
          gap: 20px;
          margin-top: 16px;
        }

        .checkbox-label {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;
          cursor: pointer;
        }

        .checkbox-label input {
          width: auto;
        }

        @media (max-width: 600px) {
          .page-header {
            flex-direction: column;
          }

          .card-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
          }

          .card-actions {
            align-self: flex-end;
          }
        }
      `}</style>
    </Layout>
  );
};

export default APIKeys;
