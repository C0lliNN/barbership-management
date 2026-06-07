'use client';

import { useEffect, useState, type ReactNode } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { apiFetch, ApiError } from '@/lib/api/client';
import { useAuth } from '@/lib/auth/session';

type GateStatus = 'checking' | 'ready';

export default function PainelLayout({ children }: { children: ReactNode }) {
  const t = useTranslations('panel');
  const router = useRouter();
  const pathname = usePathname();
  const { session, logout } = useAuth();

  const [gate, setGate] = useState<GateStatus>('checking');
  const [shopName, setShopName] = useState<string | null>(null);

  useEffect(() => {
    if (!session) {
      router.replace('/entrar');
      return;
    }

    let cancelled = false;
    apiFetch('/v1/me', { token: session.token })
      .then(() => apiFetch<{ name: string }>(`/v1/shops/${session.shopId}`, { token: session.token }))
      .then((shop) => {
        if (cancelled) return;
        setShopName(shop.name);
        setGate('ready');
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          router.replace('/entrar');
          return;
        }
        setGate('ready');
      });

    return () => {
      cancelled = true;
    };
  }, [session, router]);

  if (gate === 'checking' || !session) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6">
        <p className="text-gray-500">{t('loading')}</p>
      </main>
    );
  }

  async function handleLogout() {
    await logout();
    router.push('/entrar');
  }

  const isOwner = session.role === 'owner';
  const roleLabel = t(`roles.${session.role}`);

  return (
    <div className="flex min-h-screen flex-col">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <div>
            <p className="font-semibold text-gray-900">{shopName ?? session.shopId}</p>
            <p className="text-sm text-gray-500">
              {session.user.full_name}{' '}
              <span className="ml-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
                {roleLabel}
              </span>
            </p>
          </div>
          <button
            type="button"
            onClick={handleLogout}
            className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 transition hover:bg-gray-100"
          >
            {t('logout')}
          </button>
        </div>
        <nav className="mx-auto flex max-w-3xl gap-4 px-4 pb-3 text-sm sm:px-6">
          <Link
            href="/painel"
            className={
              pathname === '/painel'
                ? 'font-semibold text-gray-900'
                : 'text-gray-500 hover:text-gray-900'
            }
          >
            {t('nav.home')}
          </Link>
          {isOwner && (
            <Link
              href="/painel/loja"
              className={
                pathname === '/painel/loja'
                  ? 'font-semibold text-gray-900'
                  : 'text-gray-500 hover:text-gray-900'
              }
            >
              {t('nav.shopSettings')}
            </Link>
          )}
        </nav>
      </header>
      <main className="mx-auto w-full max-w-3xl flex-1 p-4 sm:p-6">{children}</main>
    </div>
  );
}
