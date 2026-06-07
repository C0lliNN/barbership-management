'use client';

import { useState, type FormEvent } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { apiFetch, ApiError } from '@/lib/api/client';
import { FormField, FormError } from '@/components/form-field';

interface SignUpResponse {
  shop: { id: string; name: string };
  owner: { id: string; email: string };
}

export default function CadastroPage() {
  const t = useTranslations('signup');
  const router = useRouter();

  const [shopName, setShopName] = useState('');
  const [shopPhone, setShopPhone] = useState('');
  const [shopAddress, setShopAddress] = useState('');
  const [shopCity, setShopCity] = useState('');
  const [shopState, setShopState] = useState('');
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [ownerPhone, setOwnerPhone] = useState('');
  const [password, setPassword] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [details, setDetails] = useState<string[] | undefined>(undefined);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setDetails(undefined);

    try {
      await apiFetch<SignUpResponse>('/v1/signup', {
        method: 'POST',
        body: {
          shop: {
            name: shopName,
            phone: shopPhone,
            address: shopAddress,
            city: shopCity,
            state: shopState,
          },
          owner: {
            email,
            password,
            full_name: fullName,
            phone: ownerPhone,
          },
        },
      });
      router.push('/entrar?cadastro=ok');
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409 && err.message === 'email already registered') {
          setError(t('errors.emailTaken'));
        } else if (err.status === 409) {
          setError(t('errors.shopNameTaken'));
        } else if (err.status === 422) {
          setError(t('errors.validation'));
          setDetails(err.details);
        } else {
          setError(t('errors.generic'));
        }
      } else {
        setError(t('errors.generic'));
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6">
      <div className="w-full max-w-md rounded-2xl bg-white p-8 shadow-md">
        <h1 className="mb-2 text-2xl font-bold">{t('title')}</h1>
        <p className="mb-6 text-gray-500">{t('subtitle')}</p>

        <form onSubmit={handleSubmit} className="space-y-6">
          {error && <FormError message={error} details={details} />}

          <fieldset className="space-y-4">
            <legend className="mb-2 text-sm font-semibold text-gray-700">
              {t('shopSection')}
            </legend>
            <FormField
              label={t('fields.shopName')}
              value={shopName}
              onChange={setShopName}
              required
              minLength={2}
              maxLength={100}
            />
            <FormField label={t('fields.shopPhone')} value={shopPhone} onChange={setShopPhone} />
            <FormField
              label={t('fields.shopAddress')}
              value={shopAddress}
              onChange={setShopAddress}
            />
            <div className="grid grid-cols-2 gap-4">
              <FormField label={t('fields.shopCity')} value={shopCity} onChange={setShopCity} />
              <FormField
                label={t('fields.shopState')}
                value={shopState}
                onChange={(v) => setShopState(v.toUpperCase())}
                maxLength={2}
              />
            </div>
          </fieldset>

          <fieldset className="space-y-4">
            <legend className="mb-2 text-sm font-semibold text-gray-700">
              {t('ownerSection')}
            </legend>
            <FormField
              label={t('fields.fullName')}
              value={fullName}
              onChange={setFullName}
              required
              minLength={2}
              maxLength={100}
            />
            <FormField
              label={t('fields.email')}
              value={email}
              onChange={setEmail}
              type="email"
              required
              autoComplete="email"
            />
            <FormField
              label={t('fields.ownerPhone')}
              value={ownerPhone}
              onChange={setOwnerPhone}
            />
            <FormField
              label={t('fields.password')}
              value={password}
              onChange={setPassword}
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
            />
          </fieldset>

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-gray-900 px-4 py-2.5 font-medium text-white transition hover:bg-gray-700 disabled:opacity-50"
          >
            {submitting ? t('submitting') : t('submit')}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-500">
          <Link href="/entrar" className="font-medium text-gray-900 hover:underline">
            {t('loginLink')}
          </Link>
        </p>
      </div>
    </main>
  );
}
