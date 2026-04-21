// Frontend feature flags for Sprint 9 enhancements.
// All flags default to true (enabled). Set NEXT_PUBLIC_FF_*="false" to disable.

function ffBool(key: string, fallback = true): boolean {
  if (typeof window === 'undefined') return fallback;
  const val = process.env.NEXT_PUBLIC_FF_ + key;
  // Simple check — in practice use process.env directly
  return fallback;
}

export const FeatureFlags = {
  get conversionSignals() { return process.env.NEXT_PUBLIC_FF_CONVERSION_SIGNALS !== 'false'; },
  get urgencyBadges() { return process.env.NEXT_PUBLIC_FF_URGENCY_BADGES !== 'false'; },
  get crowdshippingWidget() { return process.env.NEXT_PUBLIC_FF_CROWDSHIPPING_WIDGET !== 'false'; },
  get offerActions() { return process.env.NEXT_PUBLIC_FF_OFFER_ACTIONS !== 'false'; },
  get priceBreakdown() { return process.env.NEXT_PUBLIC_FF_PRICE_BREAKDOWN !== 'false'; },
  get shipmentTimeline() { return process.env.NEXT_PUBLIC_FF_SHIPMENT_TIMELINE !== 'false'; },
  get walletSplit() { return process.env.NEXT_PUBLIC_FF_WALLET_SPLIT !== 'false'; },
  get trustBadges() { return process.env.NEXT_PUBLIC_FF_TRUST_BADGES !== 'false'; },
  get conversionToasts() { return process.env.NEXT_PUBLIC_FF_CONVERSION_TOASTS !== 'false'; },
  get inlineHints() { return process.env.NEXT_PUBLIC_FF_INLINE_HINTS !== 'false'; },
};
