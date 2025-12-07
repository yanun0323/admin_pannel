import { type Component, createSignal, Show, For } from 'solid-js';
import { FiCheck, FiAlertCircle } from 'solid-icons/fi';
import Layout from '../components/Layout';
import { api } from '../lib/api';

const AccountSettings: Component = () => {
  // Change Password State
  const [currentPassword, setCurrentPassword] = createSignal('');
  const [newPassword, setNewPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [pwdIsLoading, setPwdIsLoading] = createSignal(false);
  const [pwdError, setPwdError] = createSignal('');
  const [pwdSuccess, setPwdSuccess] = createSignal('');

  // 2FA State
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [setupData, setSetupData] = createSignal<{ secret: string; qr_code: string } | null>(null);
  const [tfaPassword, setTfaPassword] = createSignal('');
  const [verificationCode, setVerificationCode] = createSignal(['', '', '', '', '', '']);
  const [tfaError, setTfaError] = createSignal<string | null>(null);
  const [tfaSuccess, setTfaSuccess] = createSignal<string | null>(null);

  let verificationInputRefs: HTMLInputElement[] = [];

  // Change Password Handlers
  const handlePasswordSubmit = async (e: Event) => {
    e.preventDefault();
    setPwdError('');
    setPwdSuccess('');

    if (!currentPassword() || !newPassword() || !confirmPassword()) {
      setPwdError('All fields are required');
      return;
    }

    if (newPassword().length < 6) {
      setPwdError('New password must be at least 6 characters');
      return;
    }

    if (newPassword() !== confirmPassword()) {
      setPwdError('New passwords do not match');
      return;
    }

    if (currentPassword() === newPassword()) {
      setPwdError('New password must be different from current password');
      return;
    }

    setPwdIsLoading(true);
    try {
      await api.changePassword(currentPassword(), newPassword());
      setPwdSuccess('Password changed successfully');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (e) {
      setPwdError(e instanceof Error ? e.message : 'Failed to change password');
    } finally {
      setPwdIsLoading(false);
    }
  };

  // 2FA Handlers
  const handleStartRebind = async (e: Event) => {
    e.preventDefault();
    if (!tfaPassword()) {
      setTfaError('Password is required');
      return;
    }

    setTfaError(null);
    setIsSubmitting(true);
    try {
      const response = await api.setupTOTPRebind(tfaPassword());
      setSetupData(response.data);
      setTfaPassword('');
    } catch (e) {
      setTfaError(e instanceof Error ? e.message : 'Failed to start 2FA rebind');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCodeInput = (index: number, value: string) => {
    const digit = value.replace(/\D/g, '').slice(-1);
    
    const newCode = [...verificationCode()];
    newCode[index] = digit;
    setVerificationCode(newCode);

    if (digit && index < 5) {
      verificationInputRefs[index + 1]?.focus();
    }

    if (newCode.every(d => d !== '') && newCode.join('').length === 6) {
      handleConfirmRebind();
    }
  };

  const handleCodeKeyDown = (index: number, e: KeyboardEvent) => {
    if (e.key === 'Backspace' && !verificationCode()[index] && index > 0) {
      verificationInputRefs[index - 1]?.focus();
    }
  };

  const handleCodePaste = (e: ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData?.getData('text') || '';
    const digits = pastedData.replace(/\D/g, '').slice(0, 6).split('');
    
    if (digits.length > 0) {
      const newCode = [...verificationCode()];
      digits.forEach((digit, i) => {
        if (i < 6) newCode[i] = digit;
      });
      setVerificationCode(newCode);
      
      const nextEmptyIndex = newCode.findIndex(d => d === '');
      const focusIndex = nextEmptyIndex === -1 ? 5 : nextEmptyIndex;
      verificationInputRefs[focusIndex]?.focus();

      if (newCode.every(d => d !== '')) {
        handleConfirmRebind();
      }
    }
  };

  const handleConfirmRebind = async () => {
    const code = verificationCode().join('');
    if (code.length !== 6) return;

    setTfaError(null);
    setIsSubmitting(true);
    try {
      await api.confirmTOTPRebind(code);
      setTfaSuccess('Two-factor authentication has been rebound successfully!');
      setSetupData(null);
      setVerificationCode(['', '', '', '', '', '']);
    } catch (e) {
      setTfaError(e instanceof Error ? e.message : 'Failed to confirm rebind');
      setVerificationCode(['', '', '', '', '', '']);
      verificationInputRefs[0]?.focus();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancelRebind = async () => {
    setTfaError(null);
    try {
      await api.cancelTOTPRebind();
      setSetupData(null);
      setVerificationCode(['', '', '', '', '', '']);
    } catch (e) {
      setTfaError(e instanceof Error ? e.message : 'Failed to cancel rebind');
    }
  };

  return (
    <Layout>
      <div class="settings-page">
        <header class="page-header">
          <h1>Account Settings</h1>
          <p>Manage your password and security settings</p>
        </header>

        <div class="settings-grid">
          {/* Change Password Column */}
          <div class="settings-column">
            <div class="section-card">
              <div class="section-header">
                <h2>Change Password</h2>
                <p>Update your account password</p>
              </div>

              <form onSubmit={handlePasswordSubmit}>
                <Show when={pwdError()}>
                  <div class="alert error">
                    <FiAlertCircle />
                    <span>{pwdError()}</span>
                  </div>
                </Show>

                <Show when={pwdSuccess()}>
                  <div class="alert success">
                    <FiCheck />
                    <span>{pwdSuccess()}</span>
                  </div>
                </Show>

                <div class="form-field">
                  <label for="current-password">Current password</label>
                  <input
                    id="current-password"
                    type="password"
                    value={currentPassword()}
                    onInput={(e) => setCurrentPassword(e.currentTarget.value)}
                    placeholder="Enter current password"
                    autocomplete="current-password"
                  />
                </div>

                <div class="form-field">
                  <label for="new-password">New password</label>
                  <input
                    id="new-password"
                    type="password"
                    value={newPassword()}
                    onInput={(e) => setNewPassword(e.currentTarget.value)}
                    placeholder="Enter new password"
                    autocomplete="new-password"
                  />
                  <span class="hint">Must be at least 6 characters</span>
                </div>

                <div class="form-field">
                  <label for="confirm-password">Confirm new password</label>
                  <input
                    id="confirm-password"
                    type="password"
                    value={confirmPassword()}
                    onInput={(e) => setConfirmPassword(e.currentTarget.value)}
                    placeholder="Confirm new password"
                    autocomplete="new-password"
                  />
                </div>

                <button type="submit" class="btn-primary" disabled={pwdIsLoading()}>
                  {pwdIsLoading() ? 'Updating...' : 'Update password'}
                </button>
              </form>
            </div>
          </div>

          {/* Two-Factor Authentication Column */}
          <div class="settings-column">
            <div class="section-card">
              <div class="section-header">
                <h2>Two-Factor Authentication</h2>
                <p>Rebind your authenticator app</p>
              </div>

              <Show when={tfaError()}>
                <div class="alert error">
                  <FiAlertCircle />
                  <span>{tfaError()}</span>
                </div>
              </Show>

              <Show when={tfaSuccess()}>
                <div class="alert success">
                  <FiCheck />
                  <span>{tfaSuccess()}</span>
                </div>
              </Show>

              <Show
                when={!setupData()}
                fallback={
                  <div class="setup-flow">
                    <h3>Scan with your new authenticator</h3>
                    <p class="setup-instruction">
                      Scan this QR code with your authenticator app
                    </p>

                    <div class="qr-container">
                      <img src={setupData()!.qr_code} alt="QR Code" class="qr-code" />
                    </div>

                    <div class="secret-container">
                      <p class="secret-label">Or enter this code manually:</p>
                      <code class="secret-code">{setupData()!.secret}</code>
                    </div>

                    <div class="verify-section">
                      <p class="verify-instruction">
                        Enter the 6-digit code from your <strong>new</strong> authenticator:
                      </p>

                      <div class="code-inputs">
                        <For each={[0, 1, 2, 3, 4, 5]}>
                          {(index) => (
                            <input
                              ref={(el) => (verificationInputRefs[index] = el)}
                              type="text"
                              inputmode="numeric"
                              maxLength={1}
                              value={verificationCode()[index]}
                              onInput={(e) => handleCodeInput(index, e.currentTarget.value)}
                              onKeyDown={(e) => handleCodeKeyDown(index, e)}
                              onPaste={handleCodePaste}
                              disabled={isSubmitting()}
                              class="code-input"
                            />
                          )}
                        </For>
                      </div>

                      <div class="setup-actions">
                        <button
                          class="btn-primary"
                          onClick={handleConfirmRebind}
                          disabled={isSubmitting() || verificationCode().join('').length !== 6}
                        >
                          {isSubmitting() ? 'Confirming...' : 'Confirm Rebind'}
                        </button>
                        <button
                          class="btn-secondary"
                          onClick={handleCancelRebind}
                          disabled={isSubmitting()}
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  </div>
                }
              >
                <div class="info-card">
                  <div class="info-icon">🔐</div>
                  <div class="info-content">
                    <h3>Rebind Authenticator</h3>
                    <p>
                      Lost access to your authenticator app? You can rebind 2FA to a new device.
                    </p>
                  </div>
                </div>

                <form class="rebind-form" onSubmit={handleStartRebind}>
                  <div class="form-field">
                    <label for="tfa-password">Enter your password to continue</label>
                    <input
                      id="tfa-password"
                      type="password"
                      value={tfaPassword()}
                      onInput={(e) => setTfaPassword(e.currentTarget.value)}
                      placeholder="Your account password"
                      disabled={isSubmitting()}
                      autocomplete="current-password"
                    />
                  </div>

                  <button
                    type="submit"
                    class="btn-primary"
                    disabled={isSubmitting() || !tfaPassword()}
                  >
                    {isSubmitting() ? 'Processing...' : 'Start Rebind Process'}
                  </button>
                </form>

                <div class="warning-card">
                  <strong>⚠️ Important:</strong>
                  <ul>
                    <li>You will need to scan a new QR code</li>
                    <li>Old authenticator codes will stop working</li>
                    <li>Make sure you have your new device ready</li>
                  </ul>
                </div>
              </Show>
            </div>
          </div>
        </div>
      </div>

      <style>{`
        .settings-page {
          max-width: 1000px;
        }

        .page-header {
          margin-bottom: 32px;
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

        .settings-grid {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 24px;
        }

        @media (max-width: 900px) {
          .settings-grid {
            grid-template-columns: 1fr;
          }
        }

        .settings-column {
          display: flex;
          flex-direction: column;
        }

        .section-card {
          background: var(--surface);
          border-radius: var(--radius-lg);
          padding: 24px;
          height: 100%;
        }

        .section-header {
          margin-bottom: 24px;
          padding-bottom: 16px;
          border-bottom: 1px solid var(--border);
        }

        .section-header h2 {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .section-header p {
          color: var(--text-secondary);
          font-size: 13px;
        }

        .alert {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 12px 14px;
          border-radius: var(--radius);
          font-size: 13px;
          margin-bottom: 20px;
        }

        .alert.error {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .alert.success {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .alert svg {
          flex-shrink: 0;
        }

        .form-field {
          margin-bottom: 16px;
        }

        .form-field label {
          display: block;
          font-size: 13px;
          font-weight: 500;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }

        .form-field input {
          width: 100%;
          padding: 10px 12px;
          background: var(--background);
          border: 1px solid var(--border);
          border-radius: var(--radius);
          color: var(--text);
          font-size: 14px;
          transition: border-color 0.15s;
        }

        .form-field input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .form-field input::placeholder {
          color: var(--text-muted);
        }

        .form-field input:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .hint {
          display: block;
          margin-top: 4px;
          font-size: 12px;
          color: var(--text-muted);
        }

        .btn-primary {
          width: 100%;
          padding: 10px 16px;
          background: var(--primary);
          border: none;
          border-radius: var(--radius);
          color: white;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: background-color 0.15s;
          margin-top: 8px;
        }

        .btn-primary:hover:not(:disabled) {
          background: var(--primary-hover);
        }

        .btn-primary:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .btn-secondary {
          width: 100%;
          padding: 10px 16px;
          background: transparent;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          color: var(--text-secondary);
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.15s;
        }

        .btn-secondary:hover:not(:disabled) {
          background: var(--surface-hover);
          color: var(--text);
        }

        .btn-secondary:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        /* 2FA Specific Styles */
        .info-card {
          display: flex;
          align-items: flex-start;
          gap: 12px;
          padding: 16px;
          background: var(--background);
          border: 1px solid var(--border);
          border-radius: var(--radius);
          margin-bottom: 20px;
        }

        .info-icon {
          font-size: 24px;
          line-height: 1;
        }

        .info-content h3 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .info-content p {
          font-size: 13px;
          color: var(--text-secondary);
          line-height: 1.4;
        }

        .rebind-form {
          margin-bottom: 20px;
        }

        .warning-card {
          padding: 14px;
          background: rgba(251, 191, 36, 0.1);
          border: 1px solid rgba(251, 191, 36, 0.3);
          border-radius: var(--radius);
          font-size: 12px;
          color: var(--text);
        }

        .warning-card strong {
          color: #f59e0b;
        }

        .warning-card ul {
          margin: 8px 0 0 0;
          padding-left: 18px;
        }

        .warning-card li {
          margin-bottom: 2px;
          color: var(--text-secondary);
        }

        .setup-flow h3 {
          font-size: 16px;
          font-weight: 600;
          margin-bottom: 8px;
        }

        .setup-instruction {
          font-size: 13px;
          color: var(--text-secondary);
          margin-bottom: 20px;
        }

        .qr-container {
          display: flex;
          justify-content: center;
          margin-bottom: 20px;
        }

        .qr-code {
          width: 160px;
          height: 160px;
          border-radius: var(--radius);
          background: white;
          padding: 8px;
        }

        .secret-container {
          text-align: center;
          margin-bottom: 20px;
          padding: 12px;
          background: var(--background);
          border-radius: var(--radius);
        }

        .secret-label {
          font-size: 12px;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }

        .secret-code {
          font-family: monospace;
          font-size: 12px;
          letter-spacing: 1px;
          color: var(--text);
          word-break: break-all;
        }

        .verify-section {
          border-top: 1px solid var(--border);
          padding-top: 20px;
        }

        .verify-instruction {
          font-size: 13px;
          color: var(--text-secondary);
          margin-bottom: 16px;
          text-align: center;
        }

        .code-inputs {
          display: flex;
          gap: 6px;
          justify-content: center;
          margin-bottom: 20px;
        }

        .code-input {
          width: 40px;
          height: 48px;
          text-align: center;
          font-size: 20px;
          font-weight: 600;
          background: var(--background);
          border: 1px solid var(--border);
          border-radius: var(--radius);
          color: var(--text);
          transition: border-color 0.15s;
        }

        .code-input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .code-input:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .setup-actions {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }
      `}</style>
    </Layout>
  );
};

export default AccountSettings;
