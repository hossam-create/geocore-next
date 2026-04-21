import { Metadata } from 'next';

export const metadata: Metadata = {
  title: {
    default: 'v Admin | mnbarh.',
    template: '%s | v Admin',
  },
  description: 'Admin dashboard for managing the marketplace platform',
  keywords: ['admin', 'dashboard', 'management', 'v', 'mnbarh.'],
  authors: [{ name: 'v' }],
  creator: 'mnbarh.',
  metadataBase: new URL('https://admin.mnbarh.com'),
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://admin.mnbarh.com',
    siteName: 'v Admin',
    title: 'v Admin | mnbarh.',
    description: 'Admin dashboard for managing the marketplace platform',
    images: [
      {
        url: '/og-image.png',
        width: 1200,
        height: 630,
        alt: 'v Admin - mnbarh.',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'v Admin | mnbarh.',
    description: 'Admin dashboard for managing the marketplace platform',
    images: ['/og-image.png'],
  },
  robots: {
    index: false,
    follow: false,
  },
};
