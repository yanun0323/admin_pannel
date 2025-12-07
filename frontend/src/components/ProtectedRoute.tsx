import { type Component, type JSX, Show } from 'solid-js';
import { Navigate } from '@solidjs/router';
import { authStore } from '../stores/auth';

interface ProtectedRouteProps {
  children: JSX.Element;
  permission?: string;
}

const ProtectedRoute: Component<ProtectedRouteProps> = (props) => {
  return (
    <Show
      when={!authStore.isLoading()}
      fallback={
        <div class="loading-container">
          <div class="loading-spinner" />
        </div>
      }
    >
      <Show
        when={authStore.isAuthenticated()}
        fallback={<Navigate href="/login" />}
      >
        <Show
          when={!props.permission || authStore.hasPermission(props.permission)}
          fallback={
            <div class="error-container">
              <h2>Access Denied</h2>
              <p>You don't have permission to view this page.</p>
            </div>
          }
        >
          {props.children}
        </Show>
      </Show>

      <style>{`
        .loading-container {
          display: flex;
          align-items: center;
          justify-content: center;
          min-height: 100vh;
        }

        .loading-spinner {
          width: 40px;
          height: 40px;
          border: 3px solid var(--surface-light);
          border-top-color: var(--primary);
          border-radius: 50%;
          animation: spin 1s linear infinite;
        }

        @keyframes spin {
          to {
            transform: rotate(360deg);
          }
        }

        .error-container {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          min-height: 60vh;
          text-align: center;
        }

        .error-container h2 {
          color: var(--danger);
          margin-bottom: 0.5rem;
        }

        .error-container p {
          color: var(--text-secondary);
        }
      `}</style>
    </Show>
  );
};

export default ProtectedRoute;
