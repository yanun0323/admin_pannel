import { A, useNavigate } from '@solidjs/router';
import { type Component, createSignal, Show, For } from 'solid-js';
import { api } from '../lib/api';

interface TOTPSetup {
  secret: string;
  qr_code: string;
}

interface PendingActivation {
  userId: number;
  totpSetup: TOTPSetup;
}

const Register: Component = () => {
  const navigate = useNavigate();
  const [username, setUsername] = createSignal('');
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [confirmPassword, setConfirmPassword] = createSignal('');
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [pendingActivation, setPendingActivation] = createSignal<PendingActivation | null>(null);
  const [activationCode, setActivationCode] = createSignal(['', '', '', '', '', '']);

  let codeInputRefs: HTMLInputElement[] = [];

  const handleRegisterSubmit = async (e: Event) => {
    e.preventDefault();
    setError(null);

    if (password() !== confirmPassword()) {
      setError('Passwords do not match');
      return;
    }

    if (password().length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await api.register(username(), email(), password());
      setPendingActivation({
        userId: response.data.user_id,
        totpSetup: response.data.totp_setup,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Registration failed');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCodeInput = (index: number, value: string) => {
    const digit = value.replace(/\D/g, '').slice(-1);
    
    const newCode = [...activationCode()];
    newCode[index] = digit;
    setActivationCode(newCode);

    if (digit && index < 5) {
      codeInputRefs[index + 1]?.focus();
    }

    if (newCode.every(d => d !== '') && newCode.join('').length === 6) {
      handleActivate();
    }
  };

  const handleCodeKeyDown = (index: number, e: KeyboardEvent) => {
    if (e.key === 'Backspace' && !activationCode()[index] && index > 0) {
      codeInputRefs[index - 1]?.focus();
    }
  };

  const handleCodePaste = (e: ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData?.getData('text') || '';
    const digits = pastedData.replace(/\D/g, '').slice(0, 6).split('');
    
    if (digits.length > 0) {
      const newCode = [...activationCode()];
      digits.forEach((digit, i) => {
        if (i < 6) newCode[i] = digit;
      });
      setActivationCode(newCode);
      
      const nextEmptyIndex = newCode.findIndex(d => d === '');
      const focusIndex = nextEmptyIndex === -1 ? 5 : nextEmptyIndex;
      codeInputRefs[focusIndex]?.focus();

      if (newCode.every(d => d !== '')) {
        handleActivate();
      }
    }
  };

  const handleActivate = async () => {
    const code = activationCode().join('');
    if (code.length !== 6) return;

    const pending = pendingActivation();
    if (!pending) return;

    setError(null);
    setIsSubmitting(true);

    try {
      await api.activateAccount(pending.userId, code);
      navigate('/login');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Activation failed');
      setActivationCode(['', '', '', '', '', '']);
      codeInputRefs[0]?.focus();
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div class="auth-container">
      <div class="auth-card">
        <Show
          when={!pendingActivation()}
          fallback={
            <>
              <div class="auth-header">
                <div class="auth-logo">Nova</div>
                <h1>Setup Two-Factor Auth</h1>
                <p>Scan the QR code with your authenticator app</p>
              </div>

              <div class="setup-content">
                <Show when={error()}>
                  <div class="alert error">{error()}</div>
                </Show>

                <div class="qr-container">
                  <img src={pendingActivation()!.totpSetup.qr_code} alt="QR Code" class="qr-code" />
                </div>

                <div class="secret-container">
                  <p class="secret-label">Or enter this code manually:</p>
                  <code class="secret-code">{pendingActivation()!.totpSetup.secret}</code>
                </div>

                <div class="verify-section">
                  <p class="verify-instruction">
                    Enter the 6-digit code to activate your account:
                  </p>

                  <div class="code-inputs">
                    <For each={[0, 1, 2, 3, 4, 5]}>
                      {(index) => (
                        <input
                          ref={(el) => (codeInputRefs[index] = el)}
                          type="text"
                          inputmode="numeric"
                          maxLength={1}
                          value={activationCode()[index]}
                          onInput={(e) => handleCodeInput(index, e.currentTarget.value)}
                          onKeyDown={(e) => handleCodeKeyDown(index, e)}
                          onPaste={handleCodePaste}
                          disabled={isSubmitting()}
                          class="code-input"
                        />
                      )}
                    </For>
                  </div>

                  <button
                    type="button"
                    class="btn-primary"
                    onClick={handleActivate}
                    disabled={isSubmitting() || activationCode().join('').length !== 6}
                  >
                    {isSubmitting() ? 'Activating...' : 'Activate Account'}
                  </button>
                </div>
              </div>
            </>
          }
        >
          <div class="auth-header">
            <div class="auth-logo">Nova</div>
            <h1>Create account</h1>
            <p>Sign up for a new account</p>
          </div>

          <form onSubmit={handleRegisterSubmit}>
            <Show when={error()}>
              <div class="alert error">{error()}</div>
            </Show>

            <div class="form-field">
              <label for="username">Username</label>
              <input
                id="username"
                type="text"
                value={username()}
                onInput={(e) => setUsername(e.currentTarget.value)}
                placeholder="Choose a username"
                required
                disabled={isSubmitting()}
                autocomplete="username"
              />
            </div>

            <div class="form-field">
              <label for="email">Email</label>
              <input
                id="email"
                type="email"
                value={email()}
                onInput={(e) => setEmail(e.currentTarget.value)}
                placeholder="Enter your email"
                required
                disabled={isSubmitting()}
                autocomplete="email"
              />
            </div>

            <div class="form-field">
              <label for="password">Password</label>
              <input
                id="password"
                type="password"
                value={password()}
                onInput={(e) => setPassword(e.currentTarget.value)}
                placeholder="Create a password"
                required
                disabled={isSubmitting()}
                autocomplete="new-password"
              />
            </div>

            <div class="form-field">
              <label for="confirm-password">Confirm password</label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword()}
                onInput={(e) => setConfirmPassword(e.currentTarget.value)}
                placeholder="Confirm your password"
                required
                disabled={isSubmitting()}
                autocomplete="new-password"
              />
            </div>

            <button type="submit" class="btn-primary" disabled={isSubmitting()}>
              {isSubmitting() ? 'Creating account...' : 'Create account'}
            </button>
          </form>

          <p class="auth-footer">
            Already have an account? <A href="/login">Sign in</A>
          </p>
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
          max-width: 400px;
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
          cursor: pointer;
        }

        .btn-primary:hover:not(:disabled) {
          background: var(--primary-hover);
        }

        .btn-primary:disabled {
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

        /* 2FA Setup Styles */
        .setup-content {
          display: flex;
          flex-direction: column;
          gap: 20px;
        }

        .qr-container {
          display: flex;
          justify-content: center;
        }

        .qr-code {
          width: 200px;
          height: 200px;
          border-radius: var(--radius);
          background: white;
          padding: 8px;
        }

        .secret-container {
          text-align: center;
          padding: 16px;
          background: var(--surface);
          border-radius: var(--radius);
        }

        .secret-label {
          font-size: 13px;
          color: var(--text-secondary);
          margin-bottom: 8px;
        }

        .secret-code {
          font-family: monospace;
          font-size: 14px;
          letter-spacing: 2px;
          color: var(--text);
          word-break: break-all;
        }

        .verify-section {
          border-top: 1px solid var(--border);
          padding-top: 20px;
        }

        .verify-instruction {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 16px;
          text-align: center;
        }

        .code-inputs {
          display: flex;
          gap: 8px;
          justify-content: center;
          margin-bottom: 20px;
        }

        .code-input {
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

        .code-input:focus {
          outline: none;
          border-color: var(--primary);
        }

        .code-input:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      `}</style>
    </div>
  );
};

export default Register;
