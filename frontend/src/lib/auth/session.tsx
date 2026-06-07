'use client';

import {
  createContext,
  useCallback,
  useContext,
  useSyncExternalStore,
  type ReactNode,
} from 'react';
import { apiFetch } from '@/lib/api/client';

export interface AuthUser {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
}

export interface AuthSession {
  token: string;
  user: AuthUser;
  shopId: string;
  role: string;
}

const STORAGE_KEY = 'barbershop.session';
const SESSION_EVENT = 'barbershop:session-change';

let cachedRaw: string | null = null;
let cachedSession: AuthSession | null = null;

function parseSession(raw: string | null): AuthSession | null {
  if (raw === cachedRaw) return cachedSession;
  cachedRaw = raw;
  try {
    cachedSession = raw ? (JSON.parse(raw) as AuthSession) : null;
  } catch {
    cachedSession = null;
  }
  return cachedSession;
}

function emitSessionChange(): void {
  window.dispatchEvent(new Event(SESSION_EVENT));
}

export function readSession(): AuthSession | null {
  if (typeof window === 'undefined') return null;
  return parseSession(window.localStorage.getItem(STORAGE_KEY));
}

export function writeSession(session: AuthSession): void {
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
  emitSessionChange();
}

export function clearSession(): void {
  window.localStorage.removeItem(STORAGE_KEY);
  emitSessionChange();
}

// Subscribes the session to localStorage so AuthProvider can stay in sync via
// useSyncExternalStore — the hydration-safe way to read an external mutable
// source (the 'storage' event alone misses same-tab writes, hence SESSION_EVENT).
function subscribe(onStoreChange: () => void): () => void {
  window.addEventListener('storage', onStoreChange);
  window.addEventListener(SESSION_EVENT, onStoreChange);
  return () => {
    window.removeEventListener('storage', onStoreChange);
    window.removeEventListener(SESSION_EVENT, onStoreChange);
  };
}

function getSnapshot(): AuthSession | null {
  return parseSession(window.localStorage.getItem(STORAGE_KEY));
}

function getServerSnapshot(): AuthSession | null {
  return null;
}

export type AuthStatus = 'authenticated' | 'unauthenticated';

interface AuthContextValue {
  session: AuthSession | null;
  status: AuthStatus;
  login: (session: AuthSession) => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const session = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const status: AuthStatus = session ? 'authenticated' : 'unauthenticated';

  const login = useCallback((next: AuthSession) => {
    writeSession(next);
  }, []);

  const logout = useCallback(async () => {
    const current = readSession();
    if (current) {
      try {
        await apiFetch('/v1/logout', { method: 'POST', token: current.token });
      } catch {
        // best-effort: still clear the local session even if the API call fails
      }
    }
    clearSession();
  }, []);

  return (
    <AuthContext.Provider value={{ session, status, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
