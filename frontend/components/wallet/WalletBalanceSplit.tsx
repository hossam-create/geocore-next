'use client';

import { formatPrice } from '@/lib/utils';
import { Wallet, Clock, ShieldCheck, Info, Upload, CreditCard, AlertTriangle } from 'lucide-react';

interface WalletBalanceSplitProps {
  available: number;
  pending: number;
  escrowed: number;
  currency?: string;
  onAddFunds?: () => void;
  onUploadProof?: () => void;
}

export function WalletBalanceSplit({ available, pending, escrowed, currency = 'AED', onAddFunds, onUploadProof }: WalletBalanceSplitProps) {
  const showPendingWarning = pending > 0 && pending > available;

  return (
    <div className="space-y-3">
      {/* Warning: pending > available */}
      {showPendingWarning && (
        <div className="flex items-center gap-2.5 rounded-xl border border-amber-300 bg-amber-50 px-4 py-3">
          <AlertTriangle size={16} className="text-amber-600 shrink-0" />
          <p className="text-sm font-medium text-amber-800">
            بعض أموالك قيد المراجعة — لن تتمكن من سحبها حتى يتم التأكيد
          </p>
        </div>
      )}

    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {/* Available */}
      <div className="bg-gradient-to-br from-[#0071CE] to-[#003f75] rounded-2xl p-6 text-white shadow-lg">
        <div className="flex items-center gap-2 text-blue-200 text-sm">
          <Wallet size={14} /> Available
        </div>
        <p className="text-4xl font-extrabold mt-2">{formatPrice(available, currency)}</p>
        <p className="text-blue-200 text-xs mt-1">Ready to withdraw or spend</p>
        {onAddFunds && (
          <button
            onClick={onAddFunds}
            className="mt-4 bg-[#FFC220] text-gray-900 font-bold text-sm px-4 py-2 rounded-xl hover:bg-yellow-400 transition-colors flex items-center gap-1.5 w-full justify-center"
          >
            <CreditCard size={14} /> Add Funds
          </button>
        )}
      </div>

      {/* Pending */}
      <div className="bg-white rounded-2xl p-5 shadow-sm border border-gray-100">
        <div className="flex items-center gap-2 text-amber-600 text-sm font-medium">
          <Clock size={14} /> Pending
        </div>
        <p className="text-2xl font-bold text-gray-900 mt-2">{formatPrice(pending, currency)}</p>
        <div className="flex items-start gap-1.5 text-xs text-gray-400 mt-1">
          <Info size={12} className="shrink-0 mt-0.5" />
          <span>Pending = waiting for payment confirmation</span>
        </div>
        {onUploadProof && pending > 0 && (
          <button
            onClick={onUploadProof}
            className="mt-3 w-full border border-amber-300 text-amber-700 font-semibold text-xs py-2 rounded-xl hover:bg-amber-50 transition-colors flex items-center justify-center gap-1.5"
          >
            <Upload size={13} /> Upload Payment Proof
          </button>
        )}
      </div>

      {/* Escrowed */}
      <div className="bg-white rounded-2xl p-5 shadow-sm border border-gray-100">
        <div className="flex items-center gap-2 text-purple-600 text-sm font-medium">
          <ShieldCheck size={14} /> Escrowed
        </div>
        <p className="text-2xl font-bold text-gray-900 mt-2">{formatPrice(escrowed, currency)}</p>
        <div className="flex items-start gap-1.5 text-xs text-gray-400 mt-1">
          <Info size={12} className="shrink-0 mt-0.5" />
          <span>Escrow = locked until delivery confirmed</span>
        </div>
      </div>
    </div>
    </div>
  );
}

interface DepositInstructionsProps {
  agentName?: string;
  agentBank?: string;
  agentAccount?: string;
  amount?: number;
  currency?: string;
  referenceCode?: string;
}

export function DepositInstructions({ agentName, agentBank, agentAccount, amount, currency = 'AED', referenceCode }: DepositInstructionsProps) {
  if (!agentName && !agentBank) return null;

  return (
    <div className="rounded-xl border border-blue-200 bg-blue-50 p-4 space-y-3">
      <p className="text-sm font-semibold text-blue-800 flex items-center gap-2">
        <CreditCard size={16} /> Deposit Instructions
      </p>
      <div className="space-y-1.5 text-sm text-blue-700">
        {agentName && <p><span className="font-medium">Agent:</span> {agentName}</p>}
        {agentBank && <p><span className="font-medium">Bank:</span> {agentBank}</p>}
        {agentAccount && <p><span className="font-medium">Account:</span> {agentAccount}</p>}
        {amount != null && <p><span className="font-medium">Amount:</span> {formatPrice(amount, currency)}</p>}
        {referenceCode && <p><span className="font-medium">Reference:</span> <code className="bg-blue-100 px-1.5 py-0.5 rounded text-xs">{referenceCode}</code></p>}
      </div>
      <p className="text-xs text-blue-500">Transfer the exact amount and upload proof below.</p>
    </div>
  );
}
