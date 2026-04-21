'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';

declare global {
  interface Window {
    google?: any;
    FB?: any;
    AppleID?: any;
  }
}

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) return resolve();
    const s = document.createElement('script');
    s.src = src;
    s.async = true;
    s.onload = () => resolve();
    s.onerror = reject;
    document.head.appendChild(s);
  });
}

async function callSocialBackend(provider: string, token: string, name?: string, email?: string) {
  const { data } = await api.post('/auth/social', { provider, token, name: name || '', email: email || '' });
  return data.data;
}

export function SocialAuthButtons({ redirectTo = '/' }: { redirectTo?: string }) {
  const [loading, setLoading] = useState<'google' | 'facebook' | 'apple' | null>(null);
  const [error, setError] = useState('');
  const router = useRouter();

  const handleGoogleLogin = async () => {
    setError('');
    setLoading('google');
    try {
      await loadScript('https://accounts.google.com/gsi/client');
      const clientId = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID;
      if (!clientId) throw new Error('Google login not configured yet.');

      const tokenClient = window.google.accounts.oauth2.initTokenClient({
        client_id: clientId,
        scope: 'email profile',
        callback: async (resp: any) => {
          if (resp.error) { setLoading(null); return; }
          try {
            const { user, access_token } = await callSocialBackend('google', resp.access_token);
            localStorage.setItem('access_token', access_token);
            localStorage.setItem('auth_user', JSON.stringify(user));
            useAuthStore.setState({ user, isAuthenticated: true });
            router.push(redirectTo);
          } catch {
            setError('فشل تسجيل الدخول بـ Google. حاول مرة أخرى.');
            setLoading(null);
          }
        },
      });
      tokenClient.requestAccessToken({ prompt: 'select_account' });
    } catch (e: any) {
      setError(e.message || 'فشل تسجيل الدخول بـ Google.');
      setLoading(null);
    }
  };

  const handleFacebookLogin = async () => {
    setError('');
    setLoading('facebook');
    try {
      const appId = process.env.NEXT_PUBLIC_FACEBOOK_APP_ID;
      if (!appId) throw new Error('Facebook login not configured yet.');

      await loadScript('https://connect.facebook.net/en_US/sdk.js');
      window.FB.init({ appId, version: 'v18.0', cookie: true, xfbml: false });
      window.FB.login(async (resp: any) => {
        if (!resp.authResponse?.accessToken) { setLoading(null); return; }
        try {
          const { user, access_token } = await callSocialBackend('facebook', resp.authResponse.accessToken);
          localStorage.setItem('access_token', access_token);
          localStorage.setItem('auth_user', JSON.stringify(user));
          useAuthStore.setState({ user, isAuthenticated: true });
          router.push(redirectTo);
        } catch {
          setError('فشل تسجيل الدخول بـ Facebook. حاول مرة أخرى.');
          setLoading(null);
        }
      }, { scope: 'email,public_profile' });
    } catch (e: any) {
      setError(e.message || 'فشل تسجيل الدخول بـ Facebook.');
      setLoading(null);
    }
  };

  const handleAppleLogin = async () => {
    setError('');
    setLoading('apple');
    try {
      const clientId = process.env.NEXT_PUBLIC_APPLE_CLIENT_ID;
      const redirectUri = (process.env.NEXT_PUBLIC_APP_URL || 'https://mnbarh.com') + '/auth/apple/callback';
      if (!clientId) throw new Error('Apple login not configured yet.');

      await loadScript('https://appleid.cdn-apple.com/appleauth/static/jsapi/appleid/1/en_US/appleid.auth.js');
      window.AppleID.auth.init({ clientId, scope: 'email name', redirectURI: redirectUri, usePopup: true });
      const appleResp = await window.AppleID.auth.signIn();
      const name = appleResp.user
        ? `${appleResp.user.name?.firstName || ''} ${appleResp.user.name?.lastName || ''}`.trim()
        : '';
      const { user, access_token } = await callSocialBackend(
        'apple',
        appleResp.authorization.id_token,
        name,
        appleResp.user?.email,
      );
      localStorage.setItem('access_token', access_token);
      localStorage.setItem('auth_user', JSON.stringify(user));
      useAuthStore.setState({ user, isAuthenticated: true });
      router.push(redirectTo);
    } catch (e: any) {
      setError(e.message || 'فشل تسجيل الدخول بـ Apple.');
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="space-y-3">
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm text-center">
          {error}
        </div>
      )}

      {/* Google */}
      <button
        type="button"
        onClick={handleGoogleLogin}
        disabled={loading !== null}
        className="w-full flex items-center justify-center gap-3 border border-gray-200 rounded-xl py-3 px-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
      >
        {loading === 'google' ? (
          <Spinner color="border-t-blue-600" />
        ) : (
          <GoogleIcon />
        )}
        Continue with Google
      </button>

      {/* Facebook */}
      <button
        type="button"
        onClick={handleFacebookLogin}
        disabled={loading !== null}
        className="w-full flex items-center justify-center gap-3 border border-gray-200 rounded-xl py-3 px-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
      >
        {loading === 'facebook' ? (
          <Spinner color="border-t-blue-700" />
        ) : (
          <FacebookIcon />
        )}
        Continue with Facebook
      </button>

      {/* Apple */}
      <button
        type="button"
        onClick={handleAppleLogin}
        disabled={loading !== null}
        className="w-full flex items-center justify-center gap-3 border border-gray-200 rounded-xl py-3 px-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
      >
        {loading === 'apple' ? (
          <Spinner color="border-t-gray-800" />
        ) : (
          <AppleIcon />
        )}
        Continue with Apple
      </button>
    </div>
  );
}

function Spinner({ color }: { color: string }) {
  return (
    <div className={`w-5 h-5 border-2 border-gray-200 ${color} rounded-full animate-spin`} />
  );
}

function GoogleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#FFC107" d="M43.611 20.083H42V20H24v8h11.303c-1.649 4.657-6.08 8-11.303 8-6.627 0-12-5.373-12-12s5.373-12 12-12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 12.955 4 4 12.955 4 24s8.955 20 20 20 20-8.955 20-20c0-1.341-.138-2.65-.389-3.917z" />
      <path fill="#FF3D00" d="m6.306 14.691 6.571 4.819C14.655 15.108 18.961 12 24 12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 16.318 4 9.656 8.337 6.306 14.691z" />
      <path fill="#4CAF50" d="M24 44c5.166 0 9.86-1.977 13.409-5.192l-6.19-5.238A11.91 11.91 0 0 1 24 36c-5.202 0-9.619-3.317-11.283-7.946l-6.522 5.025C9.505 39.556 16.227 44 24 44z" />
      <path fill="#1976D2" d="M43.611 20.083H42V20H24v8h11.303a12.04 12.04 0 0 1-4.087 5.571l.003-.002 6.19 5.238C36.971 39.205 44 34 44 24c0-1.341-.138-2.65-.389-3.917z" />
    </svg>
  );
}

function FacebookIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 48 48" aria-hidden="true">
      <linearGradient id="fbGrad" x2="0" y2="1">
        <stop offset="0" stopColor="#18AFFF" />
        <stop offset="1" stopColor="#0062DF" />
      </linearGradient>
      <path fill="url(#fbGrad)" d="M24 4C12.954 4 4 12.954 4 24s8.954 20 20 20 20-8.954 20-20S35.046 4 24 4z" />
      <path fill="#fff" d="M26.707 29.301h5.176l.813-5.258h-5.989v-2.874c0-2.184.714-4.121 2.757-4.121h3.283V12.46c-.577-.078-1.797-.248-4.102-.248-4.814 0-7.636 2.542-7.636 8.334v3.498H16.06v5.258h4.948v14.452c.98.145 1.979.247 3 .247s2.02-.102 3-.247V29.301z" />
    </svg>
  );
}

function AppleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true" fill="#111827">
      <path d="M17.05 12.54c.03 3.27 2.87 4.36 2.9 4.38-.02.08-.45 1.53-1.48 3.03-.89 1.3-1.82 2.59-3.27 2.61-1.43.03-1.89-.84-3.52-.84-1.62 0-2.13.82-3.5.87-1.4.06-2.47-1.4-3.37-2.69-1.84-2.65-3.24-7.49-1.36-10.74.94-1.62 2.61-2.64 4.42-2.67 1.38-.03 2.68.93 3.52.93.84 0 2.4-1.15 4.05-.98.69.03 2.64.28 3.89 2.1-.1.06-2.33 1.36-2.31 4.04z" />
      <path d="M14.79 4.04c.75-.91 1.26-2.18 1.12-3.44-1.08.04-2.38.72-3.15 1.63-.69.79-1.29 2.06-1.13 3.27 1.2.09 2.42-.61 3.16-1.46z" />
    </svg>
  );
}
