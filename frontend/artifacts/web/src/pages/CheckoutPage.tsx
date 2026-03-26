import { useState, useEffect } from "react";
import { useLocation, useSearch } from "wouter";
import {
  loadStripe,
  type Stripe,
  type StripeCardElement,
} from "@stripe/stripe-js";
import { Elements, CardElement, useStripe, useElements } from "@stripe/react-stripe-js";
import api from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { useAuthStore } from "@/store/auth";
import type { ApiError } from "@/lib/types";
import { ChevronLeft, Lock, CheckCircle, XCircle, CreditCard } from "lucide-react";

type CheckoutState = "form" | "processing" | "success" | "failure";

const STRIPE_PUBLISHABLE_KEY = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as string | undefined;

let stripePromise: ReturnType<typeof loadStripe> | null = null;
if (STRIPE_PUBLISHABLE_KEY) {
  stripePromise = loadStripe(STRIPE_PUBLISHABLE_KEY);
}

interface CheckoutFormProps {
  amount: number;
  currency: string;
  description: string;
  auctionId?: string;
  listingId?: string;
}

function CheckoutForm({ amount, currency, description, auctionId, listingId }: CheckoutFormProps) {
  const [, navigate] = useLocation();
  const stripe = useStripe();
  const elements = useElements();
  const [state, setState] = useState<CheckoutState>("form");
  const [errorMsg, setErrorMsg] = useState("");
  const [paymentId, setPaymentId] = useState("");
  const [cardError, setCardError] = useState("");
  const [serverAmount, setServerAmount] = useState<number | null>(null);

  const processPayment = async () => {
    setState("processing");
    setErrorMsg("");

    try {
      const intentRes = await api.post("/payments/create-payment-intent", {
        auction_id: auctionId || undefined,
        listing_id: listingId || undefined,
        currency,
      });
      const data = intentRes.data.data;
      const clientSecret: string = data.client_secret;
      const piId: string = data.payment_intent_id;
      setPaymentId(data.payment_id || "");
      if (data.amount) setServerAmount(data.amount as number);

      if (!clientSecret || !piId) {
        setState("failure");
        setErrorMsg("Stripe is not configured in this environment. No charges were made.");
        return;
      }

      if (!stripe || !elements) {
        setState("failure");
        setErrorMsg("Stripe.js failed to load. Please refresh and try again.");
        return;
      }

      const cardElement = elements.getElement(CardElement) as StripeCardElement;
      const { error, paymentIntent } = await stripe.confirmCardPayment(clientSecret, {
        payment_method: { card: cardElement },
      });

      if (error) {
        setState("failure");
        setErrorMsg(error.message || "Payment declined. Please try a different card.");
      } else if (paymentIntent?.status === "succeeded") {
        await api.post("/payments/confirm", { payment_intent_id: piId }).catch(() => {});
        setState("success");
      } else {
        setState("failure");
        setErrorMsg("Payment is still processing. You will receive a confirmation shortly.");
      }
    } catch (err) {
      const apiErr = err as ApiError;
      setState("failure");
      setErrorMsg(apiErr?.response?.data?.message ?? "Could not create payment. Please try again.");
    }
  };

  if (state === "success") {
    return (
      <div className="text-center py-12">
        <CheckCircle size={64} className="text-green-500 mx-auto mb-4" />
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Payment Successful!</h1>
        <p className="text-gray-500 text-sm mb-1">Your payment has been processed and funds are held in escrow.</p>
        {paymentId && <p className="text-xs text-gray-400 mb-6">Payment ID: {paymentId}</p>}
        <div className="flex gap-3 justify-center mt-6">
          <button
            onClick={() => navigate("/wallet")}
            className="bg-[#0071CE] text-white font-bold px-6 py-3 rounded-xl hover:bg-[#005BA1] transition-colors"
          >
            View Wallet
          </button>
          <button
            onClick={() => navigate("/")}
            className="border border-gray-200 text-gray-700 font-semibold px-6 py-3 rounded-xl hover:bg-gray-50 transition-colors"
          >
            Continue Shopping
          </button>
        </div>
      </div>
    );
  }

  if (state === "failure") {
    return (
      <div className="text-center py-12">
        <XCircle size={64} className="text-red-500 mx-auto mb-4" />
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Payment Failed</h1>
        <p className="text-gray-500 text-sm mb-6">{errorMsg}</p>
        <div className="flex gap-3 justify-center">
          <button
            onClick={() => setState("form")}
            className="bg-red-500 text-white font-bold px-6 py-3 rounded-xl hover:bg-red-600 transition-colors"
          >
            Try Again
          </button>
          <button
            onClick={() => navigate("/")}
            className="border border-gray-200 text-gray-700 font-semibold px-6 py-3 rounded-xl hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  if (state === "processing") {
    return (
      <div className="text-center py-12">
        <div className="w-16 h-16 border-4 border-[#0071CE] border-t-transparent rounded-full animate-spin mx-auto mb-4" />
        <h1 className="text-xl font-bold text-gray-900 mb-2">Processing Payment...</h1>
        <p className="text-gray-500 text-sm">Please wait while we confirm your payment.</p>
      </div>
    );
  }

  return (
    <>
      <div className="bg-white rounded-2xl shadow-sm overflow-hidden mb-5">
        <div className="bg-gradient-to-r from-[#0071CE] to-[#003f75] px-6 py-5 text-white">
          <h1 className="text-xl font-bold flex items-center gap-2">
            <Lock size={18} /> Secure Checkout
          </h1>
          <p className="text-blue-200 text-sm mt-1">Your payment is protected by Stripe</p>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <p className="text-xs text-gray-400 uppercase tracking-wide font-semibold mb-1">Order Summary</p>
            <div className="bg-gray-50 rounded-xl p-4">
              <p className="text-sm text-gray-700 font-medium">{description}</p>
              {auctionId && <p className="text-xs text-gray-400 mt-1">Auction ID: {auctionId}</p>}
              {listingId && <p className="text-xs text-gray-400 mt-1">Listing ID: {listingId}</p>}
            </div>
          </div>

          <div className="border-t border-gray-100 pt-4">
            <div className="flex items-center justify-between">
              <span className="text-gray-600 text-sm">Amount</span>
              <span className="text-2xl font-extrabold text-gray-900">{formatPrice(serverAmount ?? amount, currency)}</span>
              {serverAmount !== null && serverAmount !== amount && (
                <span className="text-xs text-amber-600 ml-1">(confirmed by server)</span>
              )}
            </div>
            <p className="text-xs text-gray-400 mt-1">Funds held in escrow until delivery confirmed</p>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-2xl shadow-sm p-6 mb-5">
        <p className="text-sm font-semibold text-gray-700 flex items-center gap-2 mb-4">
          <CreditCard size={16} className="text-[#0071CE]" /> Card Details
        </p>
        {stripePromise ? (
          <div className="border border-gray-200 rounded-xl p-4">
            <CardElement
              onChange={(e) => setCardError(e.error?.message || "")}
              options={{
                style: {
                  base: {
                    fontSize: "15px",
                    color: "#374151",
                    "::placeholder": { color: "#9CA3AF" },
                  },
                  invalid: { color: "#EF4444" },
                },
                hidePostalCode: false,
              }}
            />
            {cardError && <p className="text-xs text-red-500 mt-2">{cardError}</p>}
          </div>
        ) : (
          <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex items-center gap-3">
            <CreditCard size={20} className="text-amber-600" />
            <div>
              <p className="text-sm font-semibold text-amber-800">Stripe not configured</p>
              <p className="text-xs text-amber-700">Set VITE_STRIPE_PUBLISHABLE_KEY to enable card payments.</p>
            </div>
          </div>
        )}
      </div>

      {!stripePromise && (
        <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 text-xs text-amber-700 mb-6">
          <strong>Test Mode:</strong> No real charges will be made. In production, Stripe.js collects card details securely.
        </div>
      )}

      <button
        onClick={processPayment}
        className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        <Lock size={16} />
        {`Pay ${formatPrice(serverAmount ?? amount, currency)} Securely`}
      </button>
      <p className="text-center text-xs text-gray-400 mt-3">
        By continuing, you agree to our Terms. Funds are held in escrow until delivery is confirmed.
      </p>
    </>
  );
}

export default function CheckoutPage() {
  const [, navigate] = useLocation();
  const { isAuthenticated } = useAuthStore();
  const search = useSearch();
  const params = new URLSearchParams(search);

  const auctionId = params.get("auction_id") || undefined;
  const listingId = params.get("listing_id") || undefined;
  const amountStr = params.get("amount") || "0";
  const currency = params.get("currency") || "AED";
  const description = params.get("description") || "GeoCore Purchase";
  const amount = parseFloat(amountStr);

  if (!isAuthenticated) {
    navigate("/login?next=/checkout");
    return null;
  }

  if ((!auctionId && !listingId) || amount <= 0) {
    return (
      <div className="max-w-lg mx-auto px-4 py-16 text-center">
        <p className="text-4xl mb-4">⚠️</p>
        <h1 className="text-xl font-bold text-gray-800 mb-2">Invalid Checkout</h1>
        <p className="text-gray-500 text-sm mb-6">Missing payment details. Please return and try again.</p>
        <button onClick={() => window.history.back()} className="text-[#0071CE] hover:underline text-sm">
          ← Go Back
        </button>
      </div>
    );
  }

  const content = (
    <CheckoutForm
      amount={amount}
      currency={currency}
      description={description}
      auctionId={auctionId}
      listingId={listingId}
    />
  );

  return (
    <div className="max-w-lg mx-auto px-4 py-10">
      <button
        onClick={() => window.history.back()}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
      >
        <ChevronLeft size={16} /> Back
      </button>

      {stripePromise ? (
        <Elements stripe={stripePromise}>
          {content}
        </Elements>
      ) : (
        content
      )}
    </div>
  );
}
