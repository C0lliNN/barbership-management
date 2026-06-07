'use client';

import { useTranslations } from 'next-intl';
import { useAuth } from '@/lib/auth/session';

export default function PainelHomePage() {
  const t = useTranslations('panel');
  const { session } = useAuth();

  return (
    <div className="rounded-2xl bg-white p-8 shadow-md">
      <h1 className="mb-2 text-2xl font-bold">{t('home.title')}</h1>
      <p className="mb-6 text-gray-500">
        {t('welcome', { name: session?.user.full_name ?? '' })}
      </p>
      <p className="text-gray-400">{t('home.placeholder')}</p>
    </div>
  );
}
