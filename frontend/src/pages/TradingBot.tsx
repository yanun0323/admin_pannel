import { FiCheck, FiSave, FiX } from 'solid-icons/fi';
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
      // Deep clone the parameters
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

  // Toggle enable without confirmation dialog (real-time toggle)
  const toggleEnable = async (switcherId: string, pair: string, currentState: boolean) => {
    // Don't allow toggle while editing
    if (editingPair()) return;

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

  // Update a parameter value with nested key support
  const updateParameter = (key: string, value: any) => {
    setEditingSettings(prev => {
      const newSettings = { ...prev };
      if (key.includes('.')) {
        const [parentKey, childKey] = key.split('.');
        newSettings[parentKey] = { ...newSettings[parentKey], [childKey]: value };
      } else {
        newSettings[key] = value;
      }
      return newSettings;
    });
  };

  // Get parameter value (handles nested keys)
  const getParameterValue = (key: string, originalValue: any): any => {
    const settings = editingSettings();
    if (key.includes('.')) {
      const [parentKey, childKey] = key.split('.');
      return settings[parentKey]?.[childKey] ?? originalValue;
    }
    return settings[key] ?? originalValue;
  };

  // Render parameter display (read-only form style)
  const renderParameterDisplay = (key: string, value: any): any => {
    // Handle RESHUFFLE_ZONE specially - it's a tuple [min, max]
    if (key.endsWith('RESHUFFLE_ZONE') && Array.isArray(value) && value.length === 2) {
      return (
        <div class="param-field reshuffle-zone">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="tuple-display">
            <div class="tuple-item">
              <span class="tuple-label">Min:</span>
              <span class="tuple-value">{value[0]}</span>
            </div>
            <div class="tuple-item">
              <span class="tuple-label">Max:</span>
              <span class="tuple-value">{value[1]}</span>
            </div>
          </div>
        </div>
      );
    }

    if (typeof value === 'boolean') {
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="display-value boolean">
            <span class={`bool-badge ${value ? 'true' : 'false'}`}>
              {value ? 'Yes' : 'No'}
            </span>
          </div>
        </div>
      );
    } else if (typeof value === 'number') {
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="display-value number">{value}</div>
        </div>
      );
    } else if (Array.isArray(value)) {
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="display-value array">{JSON.stringify(value)}</div>
        </div>
      );
    } else {
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="display-value">{String(value)}</div>
        </div>
      );
    }
  };

  // Render parameter input based on type
  const renderParameterInput = (key: string, value: any): any => {
    const currentValue = getParameterValue(key, value);

    // Handle RESHUFFLE_ZONE specially - it's a tuple [min, max]
    if (key.endsWith('RESHUFFLE_ZONE') && Array.isArray(value) && value.length === 2) {
      const currentTuple = Array.isArray(currentValue) ? currentValue : value;
      return (
        <div class="param-field reshuffle-zone">
          <label>{key.replace(/^.*\./, '')}</label>
          <div class="tuple-inputs">
            <div class="tuple-input">
              <span class="tuple-label">Min:</span>
              <input
                type="number"
                step="0.0001"
                value={currentTuple[0]}
                onInput={(e) => {
                  const newValue = parseFloat(e.currentTarget.value) || 0;
                  updateParameter(key, [newValue, currentTuple[1]]);
                }}
              />
            </div>
            <div class="tuple-input">
              <span class="tuple-label">Max:</span>
              <input
                type="number"
                step="0.0001"
                value={currentTuple[1]}
                onInput={(e) => {
                  const newValue = parseFloat(e.currentTarget.value) || 0;
                  updateParameter(key, [currentTuple[0], newValue]);
                }}
              />
            </div>
          </div>
        </div>
      );
    }

    if (typeof value === 'boolean') {
      return (
        <div class="param-field checkbox-field">
          <label class="checkbox-label">
            <input
              type="checkbox"
              checked={currentValue}
              onChange={(e) => updateParameter(key, e.currentTarget.checked)}
            />
            <span>{key.replace(/^.*\./, '')}</span>
          </label>
        </div>
      );
    } else if (typeof value === 'number') {
      // Determine step based on value magnitude
      const step = Math.abs(value) < 1 ? '0.0001' : (Math.abs(value) < 100 ? '0.01' : '1');
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <input
            type="number"
            step={step}
            value={currentValue}
            onInput={(e) => updateParameter(key, parseFloat(e.currentTarget.value) || 0)}
          />
        </div>
      );
    } else if (Array.isArray(value)) {
      return (
        <div class="param-field">
          <label>{key.replace(/^.*\./, '')}</label>
          <input
            type="text"
            value={JSON.stringify(currentValue)}
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
          <label>{key.replace(/^.*\./, '')}</label>
          <input
            type="text"
            value={currentValue}
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
                    <div class="pairs-grid">
                      <For each={Object.entries(switcher.pairs)}>
                        {([pair, config]) => {
                          const setting = getSettingForPair(pair);
                          const isEditing = () =>
                            editingPair()?.switcherId === switcher.id && editingPair()?.pair === pair;

                          return (
                            <div class={`pair-card ${!config.enable ? 'disabled' : ''}`}>
                              <div class="pair-header">
                                <div class="pair-title">
                                  <h4>{pair}</h4>
                                </div>
                                <div class="pair-actions">
                                  {/* Toggle Switch */}
                                  <label class={`toggle-switch ${editingPair() ? 'disabled' : ''}`}>
                                    <input
                                      type="checkbox"
                                      checked={config.enable}
                                      onChange={() => toggleEnable(switcher.id, pair, config.enable)}
                                      disabled={!!editingPair()}
                                    />
                                    <span class="toggle-slider"></span>
                                  </label>
                                </div>
                              </div>

                              <Show when={setting}>
                                <div class="pair-body">
                                  <div class="setting-info">
                                    <span class="label">Strategy:</span>
                                    <span class="value">{setting!.strategy}</span>
                                  </div>

                                  {/* Parameters Section */}
                                  <div class="parameters-section">
                                    <div class="params-header">
                                      <h5>Parameters</h5>
                                      <Show when={!isEditing()}>
                                        <button
                                          class="btn-edit"
                                          onClick={() => startEditingPair(switcher.id, pair)}
                                        >
                                          Edit
                                        </button>
                                      </Show>
                                      <Show when={isEditing()}>
                                        <div class="edit-actions">
                                          <button class="btn-secondary" onClick={cancelEditing}>
                                            <FiX /> Cancel
                                          </button>
                                          <button class="btn-primary" onClick={() => saveSettings(pair)}>
                                            <FiSave /> Save
                                          </button>
                                        </div>
                                      </Show>
                                    </div>

                                    <Show when={isEditing()}>
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
                                    </Show>

                                    <Show when={!isEditing() && setting!.parameters}>
                                      <div class="params-grid readonly">
                                        <For each={Object.entries(setting!.parameters)}>
                                          {([key, value]) => {
                                            if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                                              return (
                                                <div class="param-group">
                                                  <h6>{key}</h6>
                                                  <For each={Object.entries(value as Record<string, any>)}>
                                                    {([subKey, subValue]) => renderParameterDisplay(`${key}.${subKey}`, subValue)}
                                                  </For>
                                                </div>
                                              );
                                            }
                                            return renderParameterDisplay(key, value);
                                          }}
                                        </For>
                                      </div>
                                    </Show>
                                  </div>
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
                    margin: 0 auto;
                }

                .page-header {
                    margin-bottom: 32px;
                }

                .page-header h1 {
                    font-size: 28px;
                    font-weight: 600;
                    margin-bottom: 4px;
                }

                .page-header p {
                    color: var(--text-secondary);
                    font-size: 14px;
                }

                .loading {
                    text-align: center;
                    padding: 40px;
                    color: var(--text-secondary);
                }

                .empty-state {
                    text-align: center;
                    padding: 60px 20px;
                    background: var(--surface);
                    border-radius: var(--radius-lg);
                    border: 1px solid var(--border);
                }

                .empty-state h3 {
                    font-size: 18px;
                    margin-bottom: 8px;
                }

                .empty-state p {
                    color: var(--text-secondary);
                }

                .switcher-card {
                    background: var(--surface);
                    border-radius: var(--radius-lg);
                    border: 1px solid var(--border);
                    margin-bottom: 24px;
                    overflow: hidden;
                }

                .switcher-header {
                    padding: 16px 20px;
                    border-bottom: 1px solid var(--border);
                }

                .switcher-header h3 {
                    font-size: 16px;
                    font-weight: 600;
                    margin: 0;
                }

                .pairs-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
                    gap: 1px;
                    background: var(--border);
                }

                .pair-card {
                    background: var(--background);
                    padding: 20px;
                }

                .pair-card.disabled {
                    opacity: 0.7;
                }

                .pair-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 16px;
                }

                .pair-title {
                    display: flex;
                    align-items: center;
                    gap: 12px;
                }

                .pair-title h4 {
                    margin: 0;
                    font-size: 16px;
                    font-weight: 600;
                }

                .status-badge {
                    padding: 4px 10px;
                    border-radius: 999px;
                    font-size: 11px;
                    font-weight: 600;
                    text-transform: uppercase;
                }

                .status-badge.enabled {
                    background: rgba(34, 197, 94, 0.15);
                    color: var(--success);
                }

                .status-badge.disabled {
                    background: rgba(239, 68, 68, 0.15);
                    color: var(--danger);
                }

                .pair-actions {
                    display: flex;
                    gap: 8px;
                    align-items: center;
                }

                /* iPhone Toggle Switch */
                .toggle-switch {
                    position: relative;
                    display: inline-block;
                    width: 50px;
                    height: 28px;
                    cursor: pointer;
                }

                .toggle-switch.disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }

                .toggle-switch input {
                    opacity: 0;
                    width: 0;
                    height: 0;
                }

                .toggle-slider {
                    position: absolute;
                    cursor: pointer;
                    top: 0;
                    left: 0;
                    right: 0;
                    bottom: 0;
                    background-color: var(--border);
                    transition: 0.3s;
                    border-radius: 28px;
                }

                .toggle-slider:before {
                    position: absolute;
                    content: "";
                    height: 22px;
                    width: 22px;
                    left: 3px;
                    bottom: 3px;
                    background-color: white;
                    transition: 0.3s;
                    border-radius: 50%;
                    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
                }

                .toggle-switch input:checked + .toggle-slider {
                    background-color: var(--success);
                }

                .toggle-switch input:checked + .toggle-slider:before {
                    transform: translateX(22px);
                }

                .toggle-switch.disabled .toggle-slider {
                    cursor: not-allowed;
                }

                .pair-body {
                    padding-top: 16px;
                    border-top: 1px solid var(--border);
                }

                .setting-info {
                    display: flex;
                    gap: 8px;
                    margin-bottom: 16px;
                }

                .setting-info .label {
                    color: var(--text-secondary);
                    font-size: 13px;
                }

                .setting-info .value {
                    font-weight: 500;
                    font-size: 13px;
                }

                .parameters-section {
                    margin-top: 16px;
                }

                .params-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 16px;
                }

                .params-header h5 {
                    font-size: 14px;
                    font-weight: 600;
                    margin: 0;
                }

                .btn-edit {
                    padding: 6px 14px;
                    font-size: 13px;
                    background: var(--primary);
                    color: white;
                    border: none;
                    border-radius: var(--radius);
                    cursor: pointer;
                    transition: all 0.15s;
                }

                .btn-edit:hover {
                    background: var(--primary-hover);
                }

                .params-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
                    gap: 16px;
                }

                .params-grid.readonly {
                    gap: 12px;
                }

                .param-group {
                    grid-column: 1 / -1;
                    background: var(--surface);
                    padding: 16px;
                    border-radius: var(--radius);
                    border: 1px solid var(--border);
                }

                .param-group h6 {
                    font-size: 13px;
                    font-weight: 600;
                    margin: 0 0 12px 0;
                    color: var(--primary);
                }

                .param-group .params-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
                    gap: 12px;
                }

                .param-field {
                    display: flex;
                    flex-direction: column;
                    gap: 6px;
                }

                .param-field label {
                    font-size: 12px;
                    font-weight: 500;
                    color: var(--text-secondary);
                }

                .param-field input {
                    padding: 8px 12px;
                    background: var(--background);
                    border: 1px solid var(--border);
                    border-radius: var(--radius);
                    color: var(--text);
                    font-size: 13px;
                }

                .param-field input:focus {
                    outline: none;
                    border-color: var(--primary);
                    box-shadow: 0 0 0 3px var(--primary-light);
                }

                .param-field.checkbox-field {
                    flex-direction: row;
                    align-items: center;
                }

                .checkbox-label {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    cursor: pointer;
                    font-size: 13px;
                }

                .checkbox-label input[type="checkbox"] {
                    width: 16px;
                    height: 16px;
                    cursor: pointer;
                }

                /* RESHUFFLE_ZONE tuple inputs */
                .param-field.reshuffle-zone {
                    grid-column: span 2;
                }

                .tuple-inputs {
                    display: flex;
                    gap: 16px;
                }

                .tuple-input {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    flex: 1;
                }

                .tuple-input .tuple-label {
                    font-size: 12px;
                    color: var(--text-secondary);
                    min-width: 30px;
                }

                .tuple-input input {
                    flex: 1;
                    padding: 8px 12px;
                    background: var(--background);
                    border: 1px solid var(--border);
                    border-radius: var(--radius);
                    color: var(--text);
                    font-size: 13px;
                }

                .tuple-display {
                    display: flex;
                    gap: 16px;
                }

                .tuple-item {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                }

                .tuple-item .tuple-label {
                    font-size: 12px;
                    color: var(--text-secondary);
                }

                .tuple-item .tuple-value {
                    font-weight: 500;
                    font-size: 13px;
                    font-family: monospace;
                }

                /* Display values (readonly) */
                .display-value {
                    padding: 8px 12px;
                    background: var(--surface);
                    border: 1px solid var(--border);
                    border-radius: var(--radius);
                    font-size: 13px;
                    font-family: monospace;
                }

                .display-value.boolean {
                    padding: 0;
                    background: transparent;
                    border: none;
                }

                .bool-badge {
                    display: inline-block;
                    padding: 4px 10px;
                    border-radius: var(--radius);
                    font-size: 12px;
                    font-weight: 500;
                }

                .bool-badge.true {
                    background: rgba(34, 197, 94, 0.15);
                    color: var(--success);
                }

                .bool-badge.false {
                    background: rgba(239, 68, 68, 0.15);
                    color: var(--danger);
                }

                .edit-actions {
                    display: flex;
                    gap: 12px;
                    justify-content: flex-end;
                }

                .btn-primary, .btn-secondary {
                    display: flex;
                    align-items: center;
                    gap: 6px;
                    padding: 6px 14px;
                    border-radius: var(--radius);
                    font-size: 13px;
                    font-weight: 500;
                    cursor: pointer;
                    transition: all 0.15s;
                    border: none;
                }

                .btn-primary {
                    background: var(--primary);
                    color: white;
                }

                .btn-primary:hover {
                    background: var(--primary-hover);
                }

                .btn-secondary {
                    background: var(--surface);
                    color: var(--text);
                    border: 1px solid var(--border);
                }

                .btn-secondary:hover {
                    background: var(--surface-hover);
                }

                .btn-icon {
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    width: 32px;
                    height: 32px;
                    border: none;
                    background: transparent;
                    color: var(--text-secondary);
                    border-radius: var(--radius);
                    cursor: pointer;
                    transition: all 0.15s;
                }

                .btn-icon:hover {
                    background: var(--surface-hover);
                    color: var(--text);
                }

                .no-setting {
                    color: var(--text-muted);
                    font-size: 13px;
                    font-style: italic;
                }

                /* Modal */
                .modal-overlay {
                    position: fixed;
                    top: 0;
                    left: 0;
                    right: 0;
                    bottom: 0;
                    background: rgba(0, 0, 0, 0.6);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 1000;
                }

                .modal {
                    background: var(--surface);
                    border-radius: var(--radius-lg);
                    border: 1px solid var(--border);
                    width: 100%;
                    max-width: 420px;
                    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
                }

                .modal-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 16px 20px;
                    border-bottom: 1px solid var(--border);
                }

                .modal-header h3 {
                    margin: 0;
                    font-size: 16px;
                    font-weight: 600;
                }

                .modal-body {
                    padding: 20px;
                }

                .modal-body p {
                    margin: 0;
                    font-size: 14px;
                    line-height: 1.6;
                }

                .modal-footer {
                    display: flex;
                    gap: 12px;
                    justify-content: flex-end;
                    padding: 16px 20px;
                    border-top: 1px solid var(--border);
                }

                @media (max-width: 768px) {
                    .pairs-grid {
                        grid-template-columns: 1fr;
                    }

                    .params-grid {
                        grid-template-columns: 1fr;
                    }

                    .tuple-inputs {
                        flex-direction: column;
                    }
                }
            `}</style>
    </Layout>
  );
};

export default TradingBot;
