'use client';
import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  Users, Copy, CheckCircle, Share2, Gift, TrendingUp,
  ArrowLeft, Loader2, ExternalLink
} from 'lucide-react';

interface ReferralCode {
  code: string;
  share_url: string;
}

interface ReferralStats {
  code: string;
  share_url: string;
  total_referrals: number;
  pending: number;
  completed: number;
  total_earned_points: number;
}

export default function ReferralPage() {
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();

  const [codeData, setCodeData] = useState<ReferralCode | null>(null);
  const [stats, setStats] = useState<ReferralStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  const [copiedURL, setCopiedURL] = useState(false);

  const fetchData = useCallback(async () => {
    if (!isAuthenticated) return;
    try {
      const [codeRes, statsRes] = await Promise.all([
        api.get('/api/v1/referral/code'),
        api.get('/api/v1/referral/stats'),
      ]);
      setCodeData(codeRes.data);
      setStats(statsRes.data.data);
    } catch (err) {
      console.error('Failed to load referral data', err);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?redirect=/referral');
      return;
    }
    fetchData();
  }, [isAuthenticated, router, fetchData]);

  const copyCode = async () => {
    if (!codeData) return;
    await navigator.clipboard.writeText(codeData.code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const copyURL = async () => {
    if (!codeData) return;
    await navigator.clipboard.writeText(codeData.share_url);
    setCopiedURL(true);
    setTimeout(() => setCopiedURL(false), 2000);
  };

  const shareNative = async () => {
    if (!codeData) return;
    if (navigator.share) {
      await navigator.share({
        title: 'Join Mnbarh — Buy & Sell with Confidence',
        text: `Use my referral code ${codeData.code} to sign up and we both earn rewards!`,
        url: codeData.share_url,
      });
    } else {
      copyURL();
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-2xl mx-auto px-4 py-8">
        {/* Back */}
        <Link href="/dashboard" className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
          <ArrowLeft className="w-4 h-4" />
          Back to Dashboard
        </Link>

        {/* Header */}
        <div className="bg-gradient-to-r from-blue-600 to-indigo-600 rounded-2xl p-8 text-white mb-6">
          <div className="flex items-center gap-3 mb-3">
            <Gift className="w-8 h-8" />
            <h1 className="text-2xl font-bold">Refer &amp; Earn</h1>
          </div>
          <p className="text-blue-100 text-sm">
            Invite friends to Mnbarh. When they complete their first order, you earn <strong>100 loyalty points</strong>.
          </p>
        </div>

        {/* Your Code */}
        <div className="bg-white rounded-2xl border border-gray-200 p-6 mb-6">
          <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Your Referral Code</h2>

          <div className="flex items-center gap-3 mb-4">
            <div className="flex-1 bg-gray-100 rounded-xl px-5 py-4 text-center">
              <span className="text-3xl font-mono font-bold tracking-widest text-gray-900">
                {codeData?.code ?? '—'}
              </span>
            </div>
            <button
              onClick={copyCode}
              className="flex items-center gap-2 px-4 py-4 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium text-sm transition-colors"
            >
              {copied ? <CheckCircle className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>

          {/* Share URL */}
          <div className="bg-gray-50 rounded-xl p-3 flex items-center gap-2 mb-4">
            <ExternalLink className="w-4 h-4 text-gray-400 flex-shrink-0" />
            <span className="text-xs text-gray-600 truncate flex-1 font-mono">{codeData?.share_url}</span>
            <button
              onClick={copyURL}
              className="text-xs text-blue-600 hover:text-blue-800 font-medium flex-shrink-0"
            >
              {copiedURL ? 'Copied!' : 'Copy link'}
            </button>
          </div>

          <button
            onClick={shareNative}
            className="w-full flex items-center justify-center gap-2 px-4 py-3 border-2 border-blue-600 text-blue-600 hover:bg-blue-50 rounded-xl font-medium transition-colors"
          >
            <Share2 className="w-4 h-4" />
            Share with friends
          </button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white rounded-2xl border border-gray-200 p-5 text-center">
            <Users className="w-6 h-6 text-blue-500 mx-auto mb-2" />
            <p className="text-2xl font-bold text-gray-900">{stats?.total_referrals ?? 0}</p>
            <p className="text-xs text-gray-500 mt-1">Total Referred</p>
          </div>
          <div className="bg-white rounded-2xl border border-gray-200 p-5 text-center">
            <TrendingUp className="w-6 h-6 text-green-500 mx-auto mb-2" />
            <p className="text-2xl font-bold text-gray-900">{stats?.completed ?? 0}</p>
            <p className="text-xs text-gray-500 mt-1">Completed</p>
          </div>
          <div className="bg-white rounded-2xl border border-gray-200 p-5 text-center">
            <Gift className="w-6 h-6 text-purple-500 mx-auto mb-2" />
            <p className="text-2xl font-bold text-gray-900">{stats?.total_earned_points ?? 0}</p>
            <p className="text-xs text-gray-500 mt-1">Points Earned</p>
          </div>
        </div>

        {/* How it works */}
        <div className="bg-white rounded-2xl border border-gray-200 p-6">
          <h2 className="font-semibold text-gray-900 mb-4">How it works</h2>
          <ol className="space-y-4">
            {[
              { step: '1', text: 'Share your unique referral code or link with friends.' },
              { step: '2', text: 'Your friend signs up using your code at registration.' },
              { step: '3', text: 'When they complete their first order, you automatically earn 100 loyalty points.' },
            ].map(({ step, text }) => (
              <li key={step} className="flex items-start gap-3">
                <span className="w-6 h-6 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center flex-shrink-0 mt-0.5">
                  {step}
                </span>
                <span className="text-sm text-gray-600">{text}</span>
              </li>
            ))}
          </ol>
          <div className="mt-4 pt-4 border-t border-gray-100">
            <Link href="/loyalty" className="text-sm text-blue-600 hover:underline font-medium">
              View your loyalty points →
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
