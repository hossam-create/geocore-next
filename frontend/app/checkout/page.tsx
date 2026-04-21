'use client'
import { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState, useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  loadStripe,
  type Stripe,
  type StripeCardElement,
} from "@stripe/stripe-js";
import { Elements, CardElement, useStripe, useElements } from "@stripe/react-stripe-js";
import api, { clearCart, getCart } from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { useAuthStore } from "@/store/auth";
import type { ApiError } from "@/lib/types";
import { ChevronLeft, Lock, CheckCircle, XCircle, CreditCard, SplitSquareHorizontal, Bitcoin } from "lucide-react";
import { CheckoutBreakdown } from "@/components/checkout/CheckoutBreakdown";
import { FeatureFlags } from "@/lib/featureFlags";
import { useTranslations } from "next-intl";

type CheckoutState = "form" | "processing" | "success" | "failure";

const STRIPE_PUBLISHABLE_KEY = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY as string | undefined;

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
  cartItems?: Array<{ listing_id: string; title: string; quantity: number; subtotal: number; currency: string }>;
  cartMode?: boolean;
  paypalToken?: string;
  paypalPayerID?: string;
  paypalCancelled?: boolean;
  cryptoStatus?: string;
}

function CheckoutForm({ amount, currency, description, auctionId, listingId, cartItems = [], cartMode = false, paypalToken, paypalPayerID, paypalCancelled = false, cryptoStatus = "" }: CheckoutFormProps) {
  const router = useRouter();
  const stripe = useStripe();
  const elements = useElements();
  const t = useTranslations("checkout");
  const [state, setState] = useState<CheckoutState>("form");
  const [errorMsg, setErrorMsg] = useState("");
  const [paymentId, setPaymentId] = useState("");
  const [cardError, setCardError] = useState("");
  const [serverAmount, setServerAmount] = useState<number | null>(null);
  const paypalHandledRef = useRef(false);

  const multipleCartItems = cartMode && cartItems.length > 1;

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

        if (cartMode) {
          await clearCart().catch(() => {});

          try {
            const ordersRes = await api.get("/orders?limit=1&page=1");
            const latestOrderID = ordersRes?.data?.data?.[0]?.id as string | undefined;
            if (latestOrderID) {
              router.push(`/orders/${latestOrderID}/success`);
              return;
            }
          } catch {
            // Fallback to orders list if latest order lookup fails.
          }

          router.push("/orders");
          return;
        }

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

  const processBNPL = async (provider: "tamara" | "tabby") => {
    setState("processing");
    setErrorMsg("");
    try {
      const returnURL = new URL(window.location.href);
      returnURL.searchParams.set("bnpl_provider", provider);
      const cancelURL = new URL(returnURL.toString());
      cancelURL.searchParams.set("bnpl_cancel", "1");

      const res = await api.post("/bnpl/create", {
        provider,
        amount: serverAmount ?? amount,
        currency,
        description,
        listing_id: listingId || undefined,
        auction_id: auctionId || undefined,
        instalments: 3,
        return_url: returnURL.toString(),
        cancel_url: cancelURL.toString(),
      });
      const data = res.data?.data ?? res.data;
      const checkoutURL: string | undefined = data?.checkout_url;
      if (!checkoutURL) {
        setState("failure");
        setErrorMsg("Could not start BNPL checkout. Please try again.");
        return;
      }
      window.location.href = checkoutURL;
    } catch (err) {
      const apiErr = err as ApiError;
      setState("failure");
      setErrorMsg(apiErr?.response?.data?.message ?? "BNPL checkout failed. Please try again.");
    }
  };

  const processPayPal = async () => {
    setState("processing");
    setErrorMsg("");

    try {
      const returnURL = new URL(window.location.href);
      returnURL.searchParams.set("paypal", "1");
      returnURL.searchParams.delete("paypal_cancel");
      returnURL.searchParams.delete("token");
      returnURL.searchParams.delete("PayerID");

      const cancelURL = new URL(returnURL.toString());
      cancelURL.searchParams.set("paypal_cancel", "1");

      const orderRes = await api.post("/payments/paypal/create", {
        auction_id: auctionId || undefined,
        listing_id: listingId || undefined,
        currency,
        description,
        return_url: returnURL.toString(),
        cancel_url: cancelURL.toString(),
      });

      const data = orderRes.data.data;
      const approvalURL: string | undefined = data?.approval_url;
      setPaymentId(data?.payment_id || "");
      if (data?.amount) setServerAmount(data.amount as number);

      if (!approvalURL) {
        setState("failure");
        setErrorMsg("Could not start PayPal checkout. Please try again.");
        return;
      }

      window.location.href = approvalURL;
    } catch (err) {
      const apiErr = err as ApiError;
      setState("failure");
      setErrorMsg(apiErr?.response?.data?.message ?? "Could not start PayPal checkout.");
    }
  };

  const processCrypto = async () => {
    setState("processing");
    setErrorMsg("");

    try {
      const returnURL = new URL(window.location.href);
      returnURL.searchParams.set("crypto", "1");
      const cancelURL = new URL(returnURL.toString());
      cancelURL.searchParams.set("crypto_cancel", "1");

      const chargeRes = await api.post("/crypto/create-charge", {
        amount: serverAmount ?? amount,
        currency,
        description,
        listing_id: listingId || undefined,
        auction_id: auctionId || undefined,
        return_url: returnURL.toString(),
        cancel_url: cancelURL.toString(),
      });

      const data = chargeRes.data?.data ?? chargeRes.data;
      const hostedURL: string | undefined = data?.hosted_url;
      if (!hostedURL) {
        setState("failure");
        setErrorMsg("Could not start crypto checkout. Please try again.");
        return;
      }

      window.location.href = hostedURL;
    } catch (err) {
      const apiErr = err as ApiError;
      setState("failure");
      setErrorMsg(apiErr?.response?.data?.message ?? "Could not start crypto checkout.");
    }
  };

  useEffect(() => {
    if (cryptoStatus === "success") {
      setState("success");
      return;
    }
    if (cryptoStatus === "cancelled") {
      setState("failure");
      setErrorMsg("Crypto checkout was cancelled.");
      return;
    }
    if (paypalCancelled) {
      setState("failure");
      setErrorMsg("PayPal checkout was cancelled.");
      return;
    }
    if (!paypalToken || !paypalPayerID || paypalHandledRef.current) {
      return;
    }

    paypalHandledRef.current = true;
    setState("processing");
    setErrorMsg("");

    (async () => {
      try {
        const captureRes = await api.post("/payments/paypal/capture", { order_id: paypalToken });
        const data = captureRes.data?.data;
        if (data?.payment_id) setPaymentId(data.payment_id as string);

        if (cartMode) {
          await clearCart().catch(() => {});
          try {
            const ordersRes = await api.get("/orders?limit=1&page=1");
            const latestOrderID = ordersRes?.data?.data?.[0]?.id as string | undefined;
            if (latestOrderID) {
              router.push(`/orders/${latestOrderID}/success`);
              return;
            }
          } catch {
            // fallback
          }
          router.push("/orders");
          return;
        }

        setState("success");
      } catch (err) {
        const apiErr = err as ApiError;
        setState("failure");
        setErrorMsg(apiErr?.response?.data?.message ?? "PayPal capture failed. Please try again.");
      }
    })();
  }, [paypalToken, paypalPayerID, paypalCancelled, cryptoStatus, cartMode, router]);

  if (state === "success") {
    return (
      <div className="text-center py-12">
        <CheckCircle size={64} className="text-green-500 mx-auto mb-4" />
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Payment Successful!</h1>
        <p className="text-gray-500 text-sm mb-1">Your payment has been processed and funds are held in escrow.</p>
        {paymentId && <p className="text-xs text-gray-400 mb-6">Payment ID: {paymentId}</p>}
        <div className="flex gap-3 justify-center mt-6">
          <button
            onClick={() => router.push("/wallet")}
            className="bg-[#0071CE] text-white font-bold px-6 py-3 rounded-xl hover:bg-[#005BA1] transition-colors"
          >
            View Wallet
          </button>
          <button
            onClick={() => router.push("/")}
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
            onClick={() => router.push("/")}
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
            <Lock size={18} /> {t("title")}
          </h1>
          <p className="text-blue-200 text-sm mt-1">Your payment is protected by Stripe</p>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <p className="text-xs text-gray-400 uppercase tracking-wide font-semibold mb-1">{t("orderSummary")}</p>
            <div className="bg-gray-50 rounded-xl p-4">
              <p className="text-sm text-gray-700 font-medium">{description}</p>
              {auctionId && <p className="text-xs text-gray-400 mt-1">Auction ID: {auctionId}</p>}
              {listingId && <p className="text-xs text-gray-400 mt-1">Listing ID: {listingId}</p>}
              {cartMode && cartItems.length > 0 && (
                <ul className="mt-3 space-y-1.5 border-t border-gray-200 pt-2">
                  {cartItems.map((item) => (
                    <li key={item.listing_id} className="flex items-center justify-between text-xs text-gray-600">
                      <span className="truncate pr-3">{item.title} × {item.quantity}</span>
                      <span className="font-medium text-gray-800">{formatPrice(item.subtotal, item.currency || currency)}</span>
                    </li>
                  ))}
                </ul>
              )}
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

      {multipleCartItems && (
        <div className="mb-4 rounded-xl border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700">
          Multi-item payment intent is not supported yet by backend pricing rules. Please keep one item in cart to continue payment.
        </div>
      )}

      {/* Sprint 9: Checkout Breakdown + Trust Signals */}
      {FeatureFlags.priceBreakdown && (
        <div className="mb-5">
          <CheckoutBreakdown
            itemPrice={amount}
            platformFee={undefined}
            total={serverAmount ?? amount}
            currency={currency}
            sellerRating={undefined}
            travelerRating={undefined}
          />
        </div>
      )}

      <button
        onClick={processPayment}
        disabled={multipleCartItems}
        className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        <Lock size={16} />
        {`Pay ${formatPrice(serverAmount ?? amount, currency)} Securely`}
      </button>
      <button
        onClick={processPayPal}
        disabled={multipleCartItems}
        className="w-full mt-3 bg-[#FFC439] hover:bg-[#f2b635] text-[#111827] font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        {`Pay with PayPal`}
      </button>

      <button
        onClick={processCrypto}
        disabled={multipleCartItems}
        className="w-full mt-3 bg-[#1E293B] hover:bg-[#0f172a] text-white font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        <Bitcoin size={16} />
        Pay with Crypto (Coinbase)
      </button>

      {/* BNPL — Tamara */}
      <button
        onClick={() => processBNPL("tamara")}
        disabled={multipleCartItems}
        className="w-full mt-3 bg-[#00C8A0] hover:bg-[#00b38d] text-white font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        <SplitSquareHorizontal size={16} />
        Pay in 3 with Tamara
      </button>

      {/* BNPL — Tabby */}
      <button
        onClick={() => processBNPL("tabby")}
        disabled={multipleCartItems}
        className="w-full mt-3 bg-[#3D3D3D] hover:bg-[#2a2a2a] text-white font-bold py-4 rounded-xl transition-colors flex items-center justify-center gap-2 text-sm"
      >
        <SplitSquareHorizontal size={16} />
        Pay in 4 with Tabby
      </button>

      <p className="text-center text-xs text-gray-400 mt-3">
        By continuing, you agree to our Terms. Funds are held in escrow until delivery is confirmed.
      </p>
    </>
  );
}

function CheckoutContent() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const params = useSearchParams();
  const fromCart = params?.get("from") === "cart";
  const paypalToken = params?.get("token") || undefined;
  const paypalPayerID = params?.get("PayerID") || undefined;
  const paypalCancelled = params?.get("paypal_cancel") === "1";
  const cryptoStatus = params?.get("crypto_status") || "";

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login?next=/checkout");
    }
  }, [isAuthenticated, router]);

  const { data: cart, isLoading: cartLoading } = useQuery({
    queryKey: ["cart", "checkout"],
    queryFn: getCart,
    enabled: isAuthenticated && fromCart,
    retry: false,
  });

  const auctionId = params?.get("auction_id") || undefined;
  const listingId = fromCart ? cart?.items?.[0]?.listing_id : (params?.get("listing_id") || undefined);
  const amountStr = params?.get("amount") || "0";
  const amount = fromCart ? (cart?.total ?? 0) : parseFloat(amountStr);
  const currency = fromCart ? (cart?.currency || cart?.items?.[0]?.currency || "AED") : (params?.get("currency") || "AED");
  const description = fromCart
    ? (cart?.items?.length
      ? `${cart.items[0].title}${cart.items.length > 1 ? ` + ${cart.items.length - 1} more item(s)` : ""}`
      : "Cart Checkout")
    : (params?.get("description") || "Mnbarh Purchase");

  if (!isAuthenticated) return null;

  if (fromCart && cartLoading) {
    return <div className="max-w-lg mx-auto px-4 py-10"><div className="h-40 animate-pulse rounded-2xl bg-gray-100" /></div>;
  }

  if ((!auctionId && !listingId) || amount <= 0 || (fromCart && (!cart || cart.items.length === 0))) {
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
      cartItems={fromCart ? (cart?.items ?? []) : []}
      cartMode={fromCart}
      paypalToken={paypalToken}
      paypalPayerID={paypalPayerID}
      paypalCancelled={paypalCancelled}
      cryptoStatus={cryptoStatus}
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

export default function CheckoutPage() {
  return (
    <Suspense fallback={null}>
      <CheckoutContent />
    </Suspense>
  );
}
