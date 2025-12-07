-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Role permissions table
CREATE TABLE IF NOT EXISTS role_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role_id INTEGER NOT NULL,
    permission TEXT NOT NULL,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    UNIQUE(role_id, permission)
);

-- User roles table
CREATE TABLE IF NOT EXISTS user_roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- Insert default admin role
INSERT OR IGNORE INTO roles (name, description) VALUES ('admin', 'Administrator with full access');
INSERT OR IGNORE INTO roles (name, description) VALUES ('user', 'Regular user with limited access');

-- Insert default permissions for admin role
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'view:dashboard' FROM roles WHERE name = 'admin';
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'view:kline' FROM roles WHERE name = 'admin';
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'manage:users' FROM roles WHERE name = 'admin';
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'manage:roles' FROM roles WHERE name = 'admin';

-- Insert default permissions for user role
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'view:dashboard' FROM roles WHERE name = 'user';
INSERT OR IGNORE INTO role_permissions (role_id, permission) 
SELECT id, 'view:kline' FROM roles WHERE name = 'user';
