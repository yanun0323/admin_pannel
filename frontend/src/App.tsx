import { Route, Router } from '@solidjs/router';
import { type Component, onMount, Show } from 'solid-js';
import ProtectedRoute from './components/ProtectedRoute';
import AccountSettings from './pages/AccountSettings';
import APIKeys from './pages/APIKeys';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import RBAC from './pages/RBAC';
import TradingBot from './pages/TradingBot';
import TradingBotMonitor from './pages/TradingBotMonitor';
import UserAdmin from './pages/UserAdmin';
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
          path="/users"
          component={() => (
            <ProtectedRoute permission="manage:users">
              <UserAdmin />
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
            <ProtectedRoute permission="manage:settings">
              <TradingBot />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/trading-bot-monitor"
          component={() => (
            <ProtectedRoute>
              <TradingBotMonitor />
            </ProtectedRoute>
          )}
        />
      </Router>
    </Show>
  );
};

export default App;
