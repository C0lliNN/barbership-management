import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';
import withPWA from '@ducanh2912/next-pwa';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  output: 'standalone',
};

export default withNextIntl(
  withPWA({ dest: 'public', disable: process.env.NODE_ENV === 'development' })(
    nextConfig
  )
);
