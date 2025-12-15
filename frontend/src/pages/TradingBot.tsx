import { FiCheck, FiEdit2, FiSave, FiX } from 'solid-icons/fi';
import { For, Show, createResource, createSignal, type Component } from 'solid-js';
import Layout from '../components/Layout';
import { api, type SettingResponse } from '../lib/api';

const TradingBot: Component = () => {
    const [switchers, { refetch: refetchSwitchers }] = createResource(async () => {
        const response = await api.listSwitchers();
        return response.data || [];
    });

    const [settings, { refetch: refetchSettings }] = createResource(async () => {
        const response = await api.listSettings();
        return response.data || [];
    });

    const [editingPair, setEditingPair] = createSignal<{ switcherId: string; pair: string } | null>(null);
    const [editingSettings, setEditingSettings] = createSignal<Record<string, any>>({});
    const [showConfirmDialog, setShowConfirmDialog] = createSignal<{
        switcherId: string;
        pair: string;
        currentState: boolean;
    } | null>(null);

    // Get setting for a specific pair
    const getSettingForPair = (pair: string): SettingResponse | undefined => {
        const [base, quote] = pair.split('_');
        return settings()?.find(s => s.base === base && s.quote === quote);
    };

    // Start editing a pair's settings
    const startEditingPair = (switcherId: string, pair: string) => {
        const setting = getSettingForPair(pair);
        if (setting) {
            setEditingPair({ switcherId, pair });
            setEditingSettings(JSON.parse(JSON.stringify(setting.parameters)));
        }
    };

    // Cancel editing
    const cancelEditing = () => {
        setEditingPair(null);
        setEditingSettings({});
    };

    // Save settings
    const saveSettings = async (pair: string) => {
        const setting = getSettingForPair(pair);
        if (!setting) return;

        try {
            await api.updateSetting(setting.id, {
                parameters: editingSettings(),
            });
            await refetchSettings();
            cancelEditing();
        } catch (e) {
            alert(e instanceof Error ? e.message : 'Failed to save settings');
        }
    };

    // Show confirmation dialog for enable toggle
    const requestToggleEnable = (switcherId: string, pair: string, currentState: boolean) => {
        setShowConfirmDialog({ switcherId, pair, currentState });
    };

    // Confirm toggle enable
    const confirmToggleEnable = async () => {
        const dialog = showConfirmDialog();
        if (!dialog) return;

        try {
            await api.updateSwitcherPair(dialog.switcherId, dialog.pair, !dialog.currentState);
            await refetchSwitchers();
            setShowConfirmDialog(null);
        } catch (e) {
            alert(e instanceof Error ? e.message : 'Failed to toggle enable state');
        }
    };

    // Update a parameter value
    const updateParameter = (key: string, value: any) => {
        setEditingSettings(prev => ({ ...prev, [key]: value }));
    };

    // Render parameter input based on type
    const renderParameterInput = (key: string, value: any) => {
        if (typeof value === 'boolean') {
            return (
                <label class="checkbox-label">
                    <input
                        type="checkbox"
                        checked={editingSettings()[key] ?? value}
                        onChange={(e) => updateParameter(key, e.currentTarget.checked)}
                    />
                    <span>{key}</span>
                </label>
            );
        } else if (typeof value === 'number') {
            return (
                <div class="param-field">
                    <label>{key}</label>
                    <input
                        type="number"
                        value={editingSettings()[key] ?? value}
                        onInput={(e) => updateParameter(key, parseFloat(e.currentTarget.value) || 0)}
                    />
                </div>
            );
        } else if (Array.isArray(value)) {
            return (
                <div class="param-field">
                    <label>{key}</label>
                    <input
                        type="text"
                        value={JSON.stringify(editingSettings()[key] ?? value)}
                        onInput={(e) => {
                            try {
                                updateParameter(key, JSON.parse(e.currentTarget.value));
                            } catch { }
                        }}
                    />
                </div>
            );
        } else {
            return (
                <div class="param-field">
                    <label>{key}</label>
                    <input
                        type="text"
                        value={editingSettings()[key] ?? value}
                        onInput={(e) => updateParameter(key, e.currentTarget.value)}
                    />
                </div>
            );
        }
    };

    return (
        <Layout>
            <div class="trading-bot-page">
                <header class="page-header">
                    <div class="header-content">
                        <h1>Trading Bot Configuration</h1>
                        <p>Manage trading pairs and their strategy settings</p>
                    </div>
                </header>

                <div class="section">
                    <Show when={!switchers.loading && !settings.loading} fallback={<div class="loading">Loading...</div>}>
                        <Show
                            when={switchers()?.length}
                            fallback={
                                <div class="empty-state">
                                    <h3>No Trading Pairs</h3>
                                    <p>No switcher configuration found</p>
                                </div>
                            }
                        >
                            <For each={switchers()}>
                                {(switcher) => (
                                    <div class="switcher-card">
                                        <div class="switcher-header">
                                            <h3>Switcher: {switcher.id}</h3>
                                        </div>
                                        <div class="pairs-grid">
                                            <For each={Object.entries(switcher.pairs)}>
                                                {([pair, config]) => {
                                                    const setting = getSettingForPair(pair);
                                                    const isEditing = editingPair()?.switcherId === switcher.id && editingPair()?.pair === pair;

                                                    return (
                                                        <div class={`pair-card ${!config.enable ? 'disabled' : ''}`}>
                                                            <div class="pair-header">
                                                                <div class="pair-title">
                                                                    <h4>{pair}</h4>
                                                                    <span class={`status-badge ${config.enable ? 'enabled' : 'disabled'}`}>
                                                                        {config.enable ? 'Enabled' : 'Disabled'}
                                                                    </span>
                                                                </div>
                                                                <div class="pair-actions">
                                                                    <Show when={!isEditing}>
                                                                        <button
                                                                            class="btn-icon"
                                                                            onClick={() => startEditingPair(switcher.id, pair)}
                                                                            title="Edit Settings"
                                                                            disabled={!setting}
                                                                        >
                                                                            <FiEdit2 />
                                                                        </button>
                                                                    </Show>
                                                                    <button
                                                                        class={`btn-toggle ${config.enable ? 'enabled' : 'disabled'}`}
                                                                        onClick={() => requestToggleEnable(switcher.id, pair, config.enable)}
                                                                        title={config.enable ? 'Disable' : 'Enable'}
                                                                    >
                                                                        {config.enable ? 'Disable' : 'Enable'}
                                                                    </button>
                                                                </div>
                                                            </div>

                                                            <Show when={setting}>
                                                                <div class="pair-body">
                                                                    <div class="setting-info">
                                                                        <span class="label">Strategy:</span>
                                                                        <span class="value">{setting!.strategy}</span>
                                                                    </div>

                                                                    <Show when={isEditing}>
                                                                        <div class="parameters-edit">
                                                                            <h5>Parameters</h5>
                                                                            <div class="params-grid">
                                                                                <For each={Object.entries(setting!.parameters)}>
                                                                                    {([key, value]) => {
                                                                                        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                                                                                            return (
                                                                                                <div class="param-group">
                                                                                                    <h6>{key}</h6>
                                                                                                    <For each={Object.entries(value as Record<string, any>)}>
                                                                                                        {([subKey, subValue]) => renderParameterInput(`${key}.${subKey}`, subValue)}
                                                                                                    </For>
                                                                                                </div>
                                                                                            );
                                                                                        }
                                                                                        return renderParameterInput(key, value);
                                                                                    }}
                                                                                </For>
                                                                            </div>
                                                                            <div class="edit-actions">
                                                                                <button class="btn-secondary" onClick={cancelEditing}>
                                                                                    <FiX /> Cancel
                                                                                </button>
                                                                                <button class="btn-primary" onClick={() => saveSettings(pair)}>
                                                                                    <FiSave /> Save
                                                                                </button>
                                                                            </div>
                                                                        </div>
                                                                    </Show>

                                                                    <Show when={!isEditing && setting!.parameters}>
                                                                        <div class="parameters-view">
                                                                            <h5>Parameters</h5>
                                                                            <div class="params-list">
                                                                                <For each={Object.entries(setting!.parameters)}>
                                                                                    {([key, value]) => (
                                                                                        <div class="param-item">
                                                                                            <span class="param-key">{key}:</span>
                                                                                            <span class="param-value">{JSON.stringify(value)}</span>
                                                                                        </div>
                                                                                    )}
                                                                                </For>
                                                                            </div>
                                                                        </div>
                                                                    </Show>
                                                                </div>
                                                            </Show>

                                                            <Show when={!setting}>
                                                                <div class="pair-body">
                                                                    <p class="no-setting">No settings configured for this pair</p>
                                                                </div>
                                                            </Show>
                                                        </div>
                                                    );
                                                }}
                                            </For>
                                        </div>
                                    </div>
                                )}
                            </For>
                        </Show>
                    </Show>
                </div>

                {/* Confirmation Dialog */}
                <Show when={showConfirmDialog()}>
                    <div class="modal-overlay" onClick={() => setShowConfirmDialog(null)}>
                        <div class="modal confirm-dialog" onClick={(e) => e.stopPropagation()}>
                            <div class="modal-header">
                                <h3>Confirm Action</h3>
                                <button class="btn-icon" onClick={() => setShowConfirmDialog(null)}>
                                    <FiX />
                                </button>
                            </div>
                            <div class="modal-body">
                                <p>
                                    Are you sure you want to <strong>{showConfirmDialog()!.currentState ? 'disable' : 'enable'}</strong> trading for{' '}
                                    <strong>{showConfirmDialog()!.pair}</strong>?
                                </p>
                            </div>
                            <div class="modal-footer">
                                <button class="btn-secondary" onClick={() => setShowConfirmDialog(null)}>
                                    Cancel
                                </button>
                                <button class="btn-primary" onClick={confirmToggleEnable}>
                                    <FiCheck /> Confirm
                                </button>
                            </div>
                        </div>
                    </div>
                </Show>
            </div>

            <style>{`
        .trading-bot-page {
          max-width: 1200px;
        }

        .page-header {
          margin-bottom: 24px;
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

        .empty-state h3 {
          font-size: 16px;
          font-weight: 500;
          margin-bottom: 8px;
        }

        .empty-state p {
          color: var(--text-secondary);
          font-size: 14px;
        }

        .switcher-card {
          margin-bottom: 24px;
        }

        .switcher-card:last-child {
          margin-bottom: 0;
        }

        .switcher-header {
          margin-bottom: 16px;
          padding-bottom: 12px;
          border-bottom: 1px solid var(--border);
        }

        .switcher-header h3 {
          font-size: 16px;
          font-weight: 500;
          color: var(--text-secondary);
        }

        .pairs-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
          gap: 16px;
        }

        .pair-card {
          border: 1px solid var(--border);
          border-radius: var(--radius);
          overflow: hidden;
          transition: all 0.2s;
        }

        .pair-card:hover {
          border-color: var(--primary);
        }

        .pair-card.disabled {
          opacity: 0.6;
        }

        .pair-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 14px 16px;
          background: var(--background);
          border-bottom: 1px solid var(--border);
        }

        .pair-title {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .pair-title h4 {
          font-size: 15px;
          font-weight: 500;
        }

        .status-badge {
          padding: 3px 10px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
        }

        .status-badge.enabled {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .status-badge.disabled {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .pair-actions {
          display: flex;
          gap: 8px;
        }

        .btn-toggle {
          padding: 6px 12px;
          border: none;
          border-radius: var(--radius);
          font-size: 12px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.15s;
        }

        .btn-toggle.enabled {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .btn-toggle.enabled:hover {
          background: rgba(239, 68, 68, 0.2);
        }

        .btn-toggle.disabled {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .btn-toggle.disabled:hover {
          background: rgba(34, 197, 94, 0.2);
        }

        .pair-body {
          padding: 16px;
        }

        .setting-info {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 12px;
        }

        .setting-info .label {
          font-size: 12px;
          color: var(--text-muted);
        }

        .setting-info .value {
          font-size: 13px;
          font-weight: 500;
        }

        .no-setting {
          color: var(--text-muted);
          font-size: 13px;
          font-style: italic;
        }

        .parameters-view h5,
        .parameters-edit h5 {
          font-size: 13px;
          font-weight: 500;
          color: var(--text-secondary);
          margin-bottom: 10px;
        }

        .params-list {
          display: flex;
          flex-direction: column;
          gap: 6px;
        }

        .param-item {
          display: flex;
          gap: 8px;
          font-size: 12px;
        }

        .param-key {
          color: var(--text-muted);
          min-width: 120px;
        }

        .param-value {
          color: var(--text);
          font-family: monospace;
          word-break: break-all;
        }

        .parameters-edit {
          margin-top: 12px;
        }

        .params-grid {
          display: flex;
          flex-direction: column;
          gap: 12px;
          margin-bottom: 16px;
        }

        .param-group {
          padding: 12px;
          background: var(--background);
          border-radius: var(--radius);
        }

        .param-group h6 {
          font-size: 12px;
          font-weight: 500;
          color: var(--text-secondary);
          margin-bottom: 8px;
        }

        .param-field {
          margin-bottom: 8px;
        }

        .param-field:last-child {
          margin-bottom: 0;
        }

        .param-field label {
          display: block;
          font-size: 12px;
          color: var(--text-muted);
          margin-bottom: 4px;
        }

        .param-field input {
          width: 100%;
          padding: 6px 10px;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: var(--surface);
          color: var(--text);
          font-size: 12px;
        }

        .param-field input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .checkbox-label {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 12px;
          cursor: pointer;
        }

        .checkbox-label input {
          width: auto;
        }

        .edit-actions {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
          margin-top: 16px;
          padding-top: 16px;
          border-top: 1px solid var(--border);
        }

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
          display: inline-flex;
          align-items: center;
          gap: 6px;
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

        .btn-icon:hover:not(:disabled) {
          color: var(--text);
          background: var(--surface-hover);
        }

        .btn-icon:disabled {
          opacity: 0.3;
          cursor: not-allowed;
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
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
          display: flex;
          flex-direction: column;
        }

        .modal.confirm-dialog {
          max-width: 400px;
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
        }

        .modal-body p {
          font-size: 14px;
          line-height: 1.6;
        }

        .modal-footer {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
          padding: 16px 20px;
          border-top: 1px solid var(--border);
        }

        @media (max-width: 768px) {
          .pairs-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
        </Layout>
    );
};

export default TradingBot;
