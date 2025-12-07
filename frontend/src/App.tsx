import { type Component, onMount, Show } from 'solid-js';
import { Router, Route } from '@solidjs/router';
import { authStore } from './stores/auth';
import ProtectedRoute from './components/ProtectedRoute';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import KLine from './pages/KLine';
import KLineSimple from './pages/KLineSimple';
import RBAC from './pages/RBAC';
import AccountSettings from './pages/AccountSettings';
import APIKeys from './pages/APIKeys';

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
          path="/kline"
          component={() => (
            <ProtectedRoute permission="view:kline">
              <KLine />
            </ProtectedRoute>
          )}
        />
        <Route
          path="/kline-simple"
          component={() => (
            <ProtectedRoute permission="view:kline">
              <KLineSimple />
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
      </Router>
    </Show>
  );
};

export default App;
