import { A, useNavigate } from '@solidjs/router';
import { type Component, createSignal, Show, For } from 'solid-js';
import { authStore } from '../stores/auth';

const Login: Component = () => {
  const navigate = useNavigate();
  const [username, setUsername] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [totpCode, setTotpCode] = createSignal(['', '', '', '', '', '']);
  const [setupCode, setSetupCode] = createSignal(['', '', '', '', '', '']);
  const [isSubmitting, setIsSubmitting] = createSignal(false);

  let totpInputRefs: HTMLInputElement[] = [];
  let setupInputRefs: HTMLInputElement[] = [];

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setIsSubmitting(true);
    authStore.setError(null);

    const result = await authStore.login(username(), password());

    if (result === 'success') {
      navigate('/');
    }
    // totp_required and totp_setup_required will show the respective UI

    setIsSubmitting(false);
  };

  const handleTotpInput = (index: number, value: string) => {
    // Only allow digits
    const digit = value.replace(/\D/g, '').slice(-1);
    
    const newCode = [...totpCode()];
    newCode[index] = digit;
    setTotpCode(newCode);

    // Auto-focus next input
    if (digit && index < 5) {
      totpInputRefs[index + 1]?.focus();
    }

    // Auto-submit when all digits are entered
    if (newCode.every(d => d !== '') && newCode.join('').length === 6) {
      handleTotpSubmit();
    }
  };

  const handleTotpKeyDown = (index: number, e: KeyboardEvent) => {
    if (e.key === 'Backspace' && !totpCode()[index] && index > 0) {
      totpInputRefs[index - 1]?.focus();
    }
  };

  const handleTotpPaste = (e: ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData?.getData('text') || '';
    const digits = pastedData.replace(/\D/g, '').slice(0, 6).split('');
    
    if (digits.length > 0) {
      const newCode = [...totpCode()];
      digits.forEach((digit, i) => {
        if (i < 6) newCode[i] = digit;
      });
      setTotpCode(newCode);
      
      // Focus the next empty input or the last one
      const nextEmptyIndex = newCode.findIndex(d => d === '');
      const focusIndex = nextEmptyIndex === -1 ? 5 : nextEmptyIndex;
      totpInputRefs[focusIndex]?.focus();

      // Auto-submit if complete
      if (newCode.every(d => d !== '')) {
        handleTotpSubmit();
      }
    }
  };

  const handleTotpSubmit = async () => {
    const code = totpCode().join('');
    if (code.length !== 6) return;

    setIsSubmitting(true);
    const success = await authStore.verifyTOTP(code);
    
    if (success) {
      navigate('/');
    } else {
      // Clear the code on error
      setTotpCode(['', '', '', '', '', '']);
      totpInputRefs[0]?.focus();
    }
    
    setIsSubmitting(false);
  };

  const handleCancelTOTP = () => {
    authStore.cancelTOTP();
    setTotpCode(['', '', '', '', '', '']);
    setPassword('');
  };

  // 2FA Setup handlers
  const handleSetupInput = (index: number, value: string) => {
    const digit = value.replace(/\D/g, '').slice(-1);
    
    const newCode = [...setupCode()];
    newCode[index] = digit;
    setSetupCode(newCode);

    if (digit && index < 5) {
      setupInputRefs[index + 1]?.focus();
    }

    if (newCode.every(d => d !== '') && newCode.join('').length === 6) {
      handleSetupSubmit();
    }
  };

  const handleSetupKeyDown = (index: number, e: KeyboardEvent) => {
    if (e.key === 'Backspace' && !setupCode()[index] && index > 0) {
      setupInputRefs[index - 1]?.focus();
    }
  };

  const handleSetupPaste = (e: ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData?.getData('text') || '';
    const digits = pastedData.replace(/\D/g, '').slice(0, 6).split('');
    
    if (digits.length > 0) {
      const newCode = [...setupCode()];
      digits.forEach((digit, i) => {
        if (i < 6) newCode[i] = digit;
      });
      setSetupCode(newCode);
      
      const nextEmptyIndex = newCode.findIndex(d => d === '');
      const focusIndex = nextEmptyIndex === -1 ? 5 : nextEmptyIndex;
      setupInputRefs[focusIndex]?.focus();

      if (newCode.every(d => d !== '')) {
        handleSetupSubmit();
      }
    }
  };

  const handleSetupSubmit = async () => {
    const code = setupCode().join('');
    if (code.length !== 6) return;

    setIsSubmitting(true);
    const success = await authStore.activateTOTP(code);
    
    if (success) {
      // Now pendingTOTP is set, show TOTP verification screen
      setSetupCode(['', '', '', '', '', '']);
    } else {
      setSetupCode(['', '', '', '', '', '']);
      setupInputRefs[0]?.focus();
    }
    
    setIsSubmitting(false);
  };

  const handleCancelSetup = () => {
    authStore.cancelTOTPSetup();
    setSetupCode(['', '', '', '', '', '']);
    setPassword('');
  };

  return (
    <div class="auth-container">
      <div class="auth-card">
        <Show
          when={!authStore.pendingTOTPSetup()}
          fallback={
            <>
              <div class="auth-header">
                <div class="auth-logo">Nova</div>
                <h1>Setup Two-Factor Authentication</h1>
                <p>Scan the QR code with your authenticator app</p>
              </div>

              <div class="setup-form">
                <Show when={authStore.error()}>
                  <div class="alert error">{authStore.error()}</div>
                </Show>

                <div class="qr-container">
                  <img src={authStore.pendingTOTPSetup()!.qrCode} alt="QR Code" class="qr-code" />
                </div>

                <div class="secret-container">
                  <p class="secret-label">Or enter this code manually:</p>
                  <code class="secret-code">{authStore.pendingTOTPSetup()!.secret}</code>
                </div>

                <div class="verify-section">
                  <p class="verify-label">Enter the 6-digit code to verify:</p>
                  <div class="totp-inputs">
                    <For each={[0, 1, 2, 3, 4, 5]}>
                      {(index) => (
                        <input
                          ref={(el) => (setupInputRefs[index] = el)}
                          type="text"
                          inputmode="numeric"
                          maxLength={1}
                          value={setupCode()[index]}
                          onInput={(e) => handleSetupInput(index, e.currentTarget.value)}
                          onKeyDown={(e) => handleSetupKeyDown(index, e)}
                          onPaste={handleSetupPaste}
                          disabled={isSubmitting()}
                          class="totp-input"
                          autocomplete="one-time-code"
                        />
                      )}
                    </For>
                  </div>
                </div>

                <button
                  type="button"
                  class="btn-primary"
                  onClick={handleSetupSubmit}
                  disabled={isSubmitting() || setupCode().join('').length !== 6}
                >
                  {isSubmitting() ? 'Activating...' : 'Activate Account'}
                </button>

                <button
                  type="button"
                  class="btn-secondary"
                  onClick={handleCancelSetup}
                  disabled={isSubmitting()}
                >
                  Back to login
                </button>
              </div>
            </>
          }
        >
        <Show
          when={!authStore.pendingTOTP()}
          fallback={
            <>
              <div class="auth-header">
                <div class="auth-logo">Nova</div>
                <h1>Two-Factor Authentication</h1>
                <p>Enter the 6-digit code from your authenticator app</p>
              </div>

              <div class="totp-form">
                <Show when={authStore.error()}>
                  <div class="alert error">{authStore.error()}</div>
                </Show>

                <div class="totp-inputs">
                  <For each={[0, 1, 2, 3, 4, 5]}>
                    {(index) => (
                      <input
                        ref={(el) => (totpInputRefs[index] = el)}
                        type="text"
                        inputmode="numeric"
                        maxLength={1}
                        value={totpCode()[index]}
                        onInput={(e) => handleTotpInput(index, e.currentTarget.value)}
                        onKeyDown={(e) => handleTotpKeyDown(index, e)}
                        onPaste={handleTotpPaste}
                        disabled={isSubmitting()}
                        class="totp-input"
                        autocomplete="one-time-code"
                      />
                    )}
                  </For>
                </div>

                <button
                  type="button"
                  class="btn-primary"
                  onClick={handleTotpSubmit}
                  disabled={isSubmitting() || totpCode().join('').length !== 6}
                >
                  {isSubmitting() ? 'Verifying...' : 'Verify'}
                </button>

                <button
                  type="button"
                  class="btn-secondary"
                  onClick={handleCancelTOTP}
                  disabled={isSubmitting()}
                >
                  Back to login
                </button>
              </div>
            </>
          }
        >
          <div class="auth-header">
            <div class="auth-logo">Nova</div>
            <h1>Welcome back</h1>
            <p>Sign in to your account</p>
          </div>

          <form onSubmit={handleSubmit}>
            <Show when={authStore.error()}>
              <div class="alert error">{authStore.error()}</div>
            </Show>

            <div class="form-field">
              <label for="username">Username</label>
              <input
                id="username"
                type="text"
                value={username()}
                onInput={(e) => setUsername(e.currentTarget.value)}
                placeholder="Enter username"
                required
                disabled={isSubmitting()}
                autocomplete="username"
              />
            </div>

            <div class="form-field">
              <label for="password">Password</label>
              <input
                id="password"
                type="password"
                value={password()}
                onInput={(e) => setPassword(e.currentTarget.value)}
                placeholder="Enter password"
                required
                disabled={isSubmitting()}
                autocomplete="current-password"
              />
            </div>

            <button type="submit" class="btn-primary" disabled={isSubmitting()}>
              {isSubmitting() ? 'Signing in...' : 'Sign in'}
            </button>
          </form>

          <p class="auth-footer">
            Don't have an account? <A href="/register">Create one</A>
          </p>
        </Show>
        </Show>
      </div>

      <style>{`
        .auth-container {
          display: flex;
          align-items: center;
          justify-content: center;
          min-height: 100vh;
          padding: 20px;
        }

        .auth-card {
          width: 100%;
          max-width: 360px;
        }

        .auth-header {
          text-align: center;
          margin-bottom: 32px;
        }

        .auth-logo {
          width: 48px;
          height: 48px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
          background: var(--primary);
          color: white;
          font-weight: 600;
          font-size: 16px;
          border-radius: var(--radius-lg);
          margin-bottom: 20px;
        }

        .auth-header h1 {
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 8px;
        }

        .auth-header p {
          color: var(--text-secondary);
          font-size: 14px;
        }

        .alert {
          padding: 12px 14px;
          border-radius: var(--radius);
          font-size: 13px;
          margin-bottom: 20px;
        }

        .alert.error {
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
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
          background: var(--surface);
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

        .form-field input:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .form-field input::placeholder {
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
          transition: all 0.15s;
          margin-top: 12px;
        }

        .btn-secondary:hover:not(:disabled) {
          background: var(--surface);
          color: var(--text);
        }

        .btn-secondary:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .totp-form {
          display: flex;
          flex-direction: column;
          gap: 20px;
        }

        .totp-inputs {
          display: flex;
          gap: 8px;
          justify-content: center;
        }

        .totp-input {
          width: 48px;
          height: 56px;
          text-align: center;
          font-size: 24px;
          font-weight: 600;
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: var(--radius);
          color: var(--text);
          transition: border-color 0.15s;
        }

        .totp-input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .totp-input:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .auth-footer {
          text-align: center;
          margin-top: 24px;
          font-size: 13px;
          color: var(--text-secondary);
        }

        .auth-footer a {
          color: var(--primary);
          font-weight: 500;
        }

        .setup-form {
          display: flex;
          flex-direction: column;
          gap: 20px;
        }

        .qr-container {
          display: flex;
          justify-content: center;
        }

        .qr-code {
          width: 180px;
          height: 180px;
          border-radius: var(--radius);
          background: white;
          padding: 8px;
        }

        .secret-container {
          text-align: center;
          padding: 12px;
          background: var(--surface);
          border-radius: var(--radius);
        }

        .secret-label {
          font-size: 12px;
          color: var(--text-secondary);
          margin-bottom: 6px;
        }

        .secret-code {
          font-family: monospace;
          font-size: 13px;
          letter-spacing: 1px;
          color: var(--text);
          word-break: break-all;
        }

        .verify-section {
          text-align: center;
        }

        .verify-label {
          font-size: 13px;
          color: var(--text-secondary);
          margin-bottom: 12px;
        }
      `}</style>
    </div>
  );
};

export default Login;
