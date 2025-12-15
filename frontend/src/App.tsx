import { Route, Router } from '@solidjs/router';
import { type Component, onMount, Show } from 'solid-js';
import ProtectedRoute from './components/ProtectedRoute';
import AccountSettings from './pages/AccountSettings';
import APIKeys from './pages/APIKeys';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import RBAC from './pages/RBAC';
import Register from './pages/Register';
import TradingBot from './pages/TradingBot';
import { authStore } from './stores/auth';

const App: Component = () => {
  onMount(() => {
    authStore.initialize();
  });

  return (
    <Show
      when={!authStore.isLoading()}
      fallback={
        <div class="app-loading">
          <div class="loading-spinner" />
        </div>
      }
    >
      <Router>
        <Route path="/login" component={Login} />
        <Route path="/register" component={Register} />
        <Route
          path="/"
          component={() => (
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/rbac"
          component={() => (
            <ProtectedRoute permission="manage:roles">
              <RBAC />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/settings"
          component={() => (
            <ProtectedRoute>
              <AccountSettings />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/api-keys"
          component={() => (
            <ProtectedRoute permission="manage:api_keys">
              <APIKeys />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/trading-bot"
          component={() => (
            <ProtectedRoute permission="manage:switchers">
              <TradingBot />
            </ProtectedRoute>
          )}
        />
      </Router>
    </Show>
  );
};

export default App;
