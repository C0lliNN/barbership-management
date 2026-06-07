'use client';

import { useEffect, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { apiFetch, ApiError } from '@/lib/api/client';
import { useAuth } from '@/lib/auth/session';
import { FormField, FormError, FormSuccess } from '@/components/form-field';

interface ShopProfile {
  id: string;
  name: string;
  slug: string;
  phone?: string;
  address?: string;
  city?: string;
  state?: string;
  updated_at: string;
}

type LoadStatus = 'loading' | 'ready' | 'error';

export default function LojaSettingsPage() {
  const t = useTranslations('shopSettings');
  const router = useRouter();
  const { session } = useAuth();

  const [load, setLoad] = useState<LoadStatus>('loading');
  const [phone, setPhone] = useState('');
  const [address, setAddress] = useState('');
  const [city, setCity] = useState('');
  const [state, setState] = useState('');

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [details, setDetails] = useState<string[] | undefined>(undefined);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    if (!session) return;
    if (session.role !== 'owner') {
      router.replace('/painel');
      return;
    }

    let cancelled = false;
    apiFetch<ShopProfile>(`/v1/shops/${session.shopId}`, { token: session.token })
      .then((shop) => {
        if (cancelled) return;
        setPhone(shop.phone ?? '');
        setAddress(shop.address ?? '');
        setCity(shop.city ?? '');
        setState(shop.state ?? '');
        setLoad('ready');
      })
      .catch(() => {
        if (!cancelled) setLoad('error');
      });

    return () => {
      cancelled = true;
    };
  }, [session, router]);

  if (!session || session.role !== 'owner') {
    return null;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!session) return;
    setSaving(true);
    setError(null);
    setDetails(undefined);
    setSuccess(false);

    try {
      const updated = await apiFetch<ShopProfile>(`/v1/shops/${session.shopId}`, {
        method: 'PATCH',
        token: session.token,
        body: { phone, address, city, state },
      });
      setPhone(updated.phone ?? '');
      setAddress(updated.address ?? '');
      setCity(updated.city ?? '');
      setState(updated.state ?? '');
      setSuccess(true);
    } catch (err) {
      if (err instanceof ApiError && err.status === 422) {
        setError(t('errors.validation'));
        setDetails(err.details);
      } else {
        setError(t('errors.generic'));
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="rounded-2xl bg-white p-8 shadow-md">
      <h1 className="mb-2 text-2xl font-bold">{t('title')}</h1>
      <p className="mb-6 text-gray-500">{t('subtitle')}</p>

      {load === 'loading' && <p className="text-gray-500">{t('saving')}</p>}
      {load === 'error' && <FormError message={t('errors.generic')} />}

      {load !== 'loading' && load !== 'error' && (
        <form onSubmit={handleSubmit} className="space-y-4">
          {success && <FormSuccess message={t('success')} />}
          {error && <FormError message={error} details={details} />}

          <FormField label={t('fields.phone')} value={phone} onChange={setPhone} />
          <FormField label={t('fields.address')} value={address} onChange={setAddress} />
          <div className="grid grid-cols-2 gap-4">
            <FormField label={t('fields.city')} value={city} onChange={setCity} />
            <FormField
              label={t('fields.state')}
              value={state}
              onChange={(v) => setState(v.toUpperCase())}
              maxLength={2}
            />
          </div>

          <button
            type="submit"
            disabled={saving}
            className="w-full rounded-lg bg-gray-900 px-4 py-2.5 font-medium text-white transition hover:bg-gray-700 disabled:opacity-50 sm:w-auto"
          >
            {saving ? t('saving') : t('save')}
          </button>
        </form>
      )}
    </div>
  );
}
