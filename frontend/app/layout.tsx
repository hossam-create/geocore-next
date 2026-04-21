import { NextIntlClientProvider } from 'next-intl';
import { getLocale, getMessages } from 'next-intl/server';
import { Header } from '@/components/layout/Header';
import { Footer } from '@/components/layout/Footer';
import { Providers } from './providers';
import './globals.css';
import { Metadata } from 'next';
import { ConversionToastContainer } from '@/components/ui/ConversionToast';

export const metadata: Metadata = {
  title: {
    default: 'mnbarh',
    template: '%s | mnbarh',
  },
  description: 'منصة الإعلانات المبوبة والمزادات. بيع واشتري سيارات، إلكترونيات، عقارات والمزيد.',
  keywords: ['إعلانات مبوبة', 'مزادات', 'بيع وشراء', 'سوق', 'من بره', 'mnbarh'],
  authors: [{ name: 'من بره' }],
  creator: 'mnbarh',
  metadataBase: new URL('https://mnbarh.com'),
  openGraph: {
    type: 'website',
    locale: 'ar_EG',
    url: 'https://mnbarh.com',
    siteName: 'من بره',
    title: 'من بره | mnbarh',
    description: 'منصة الإعلانات المبوبة والمزادات.',
    images: [
      {
        url: '/og-image.png',
        width: 1200,
        height: 630,
        alt: 'من بره - mnbarh',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'من بره | mnbarh',
    description: 'منصة الإعلانات المبوبة والمزادات.',
    images: ['/og-image.png'],
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
    },
  },
  icons: {
    icon: '/favicon.ico',
    shortcut: '/favicon.ico',
    apple: '/apple-touch-icon.png',
  },
  manifest: '/site.webmanifest',
  appleWebApp: {
    capable: true,
    title: 'من بره',
    statusBarStyle: 'default',
    startupImage: '/splash-screen.png',
  },
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const locale = await getLocale();
  const messages = await getMessages();
  const dir = locale === 'ar' ? 'rtl' : 'ltr';

  return (
    <html lang={locale} dir={dir}>
      <body>
        <NextIntlClientProvider messages={messages}>
          <Providers>
            <div className="min-h-screen flex flex-col">
              <Header />
              <main className="flex-1">
                {children}
              </main>
              <Footer />
            </div>
            <ConversionToastContainer />
          </Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
