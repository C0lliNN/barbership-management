import { getTranslations } from 'next-intl/server';

async function getApiHealth(): Promise<'online' | 'offline' | 'error'> {
  const apiUrl =
    process.env.API_INTERNAL_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    'http://localhost:8080';
  try {
    const res = await fetch(`${apiUrl}/health`, {
      next: { revalidate: 10 },
    });
    return res.ok ? 'online' : 'offline';
  } catch {
    return 'error';
  }
}

export default async function HomePage() {
  const t = await getTranslations('landing');
  const health = await getApiHealth();

  const statusColor =
    health === 'online'
      ? 'text-green-600'
      : health === 'offline'
        ? 'text-red-600'
        : 'text-yellow-600';

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 text-center shadow-md">
        <div className="mb-4 text-5xl">💈</div>
        <h1 className="mb-2 text-2xl font-bold">{t('title')}</h1>
        <p className="mb-6 text-gray-500">{t('subtitle')}</p>
        <span
          className={`rounded-full bg-gray-100 px-4 py-1.5 text-sm font-medium ${statusColor}`}
        >
          {t(`apiStatus.${health}`)}
        </span>
      </div>
    </main>
  );
}
