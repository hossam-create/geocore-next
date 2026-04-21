import Link from 'next/link';
import { useTranslations } from 'next-intl';

export default function NotFound() {
  const t = useTranslations('errors');
  return (
    <div className="min-h-[60vh] flex items-center justify-center text-center">
      <div>
        <p className="text-6xl mb-4">404</p>
        <h1 className="text-2xl font-bold text-gray-800">{t('pageNotFound')}</h1>
        <Link href="/" className="mt-4 text-[#0071CE] hover:underline block">
          ← {t('goHome')}
        </Link>
      </div>
    </div>
  );
}
