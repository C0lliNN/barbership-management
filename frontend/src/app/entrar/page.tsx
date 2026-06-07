'use client';

import { Suspense, useState, type FormEvent } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { apiFetch, ApiError } from '@/lib/api/client';
import { useAuth, type AuthUser } from '@/lib/auth/session';
import { FormField, FormError, FormSuccess } from '@/components/form-field';

interface LoginResponse {
  token: string;
  user: AuthUser;
  shop_id: string;
  role: string;
}

export default function EntrarPage() {
  return (
    <Suspense>
      <EntrarForm />
    </Suspense>
  );
}

function EntrarForm() {
  const t = useTranslations('login');
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const showSignupSuccess = searchParams.get('cadastro') === 'ok';

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const resp = await apiFetch<LoginResponse>('/v1/login', {
        method: 'POST',
        body: { email, password },
      });
      login({ token: resp.token, user: resp.user, shopId: resp.shop_id, role: resp.role });
      router.push('/painel');
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError(t('errors.invalidCredentials'));
      } else {
        setError(t('errors.generic'));
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 shadow-md">
        <h1 className="mb-2 text-2xl font-bold">{t('title')}</h1>
        <p className="mb-6 text-gray-500">{t('subtitle')}</p>

        <form onSubmit={handleSubmit} className="space-y-4">
          {showSignupSuccess && <FormSuccess message={t('signupSuccess')} />}
          {error && <FormError message={error} />}

          <FormField
            label={t('fields.email')}
            value={email}
            onChange={setEmail}
            type="email"
            required
            autoComplete="email"
          />
          <FormField
            label={t('fields.password')}
            value={password}
            onChange={setPassword}
            type="password"
            required
            autoComplete="current-password"
          />

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-gray-900 px-4 py-2.5 font-medium text-white transition hover:bg-gray-700 disabled:opacity-50"
          >
            {submitting ? t('submitting') : t('submit')}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-500">
          <Link href="/cadastro" className="font-medium text-gray-900 hover:underline">
            {t('signupLink')}
          </Link>
        </p>
      </div>
    </main>
  );
}
