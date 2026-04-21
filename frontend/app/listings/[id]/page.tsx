'use client'
import Link from 'next/link';
import { useRouter, useParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useEffect, useRef, useCallback } from "react";
import api, { addCartItem, addWatchlistItem, removeWatchlistItem, removeWatchlistPriceSnapshot, setWatchlistPriceSnapshot } from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { getCategorySchema, formatFieldValue } from "@/lib/categoryFields";
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { useAuthStore } from "@/store/auth";
import { Heart, Star, MessageCircle, Share2, ChevronLeft, ChevronRight, TrendingDown, Layers, Trophy, Store, ZoomIn, Truck, RotateCcw, Shield, ThumbsUp, HelpCircle, Send, Clock } from "lucide-react";
import { getAuctionType, AUCTION_TYPE_BADGE } from "@/lib/auctionTypes";
import { SimilarListings } from "@/components/listings/SimilarListings";
import { ALL_LISTINGS } from "@/lib/recommendations";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { ConversionSignals } from "@/components/listings/ConversionSignals";
import { UrgencyBadge } from "@/components/listings/UrgencyBadge";
import { CrowdshippingWidget } from "@/components/listings/CrowdshippingWidget";
import { OfferActions } from "@/components/listings/OfferActions";
import { PriceBreakdown } from "@/components/listings/PriceBreakdown";
import { TrustBadges } from "@/components/trust/TrustBadges";
import { InlineHint } from "@/components/ui/InlineHint";
import { FeatureFlags } from "@/lib/featureFlags";

const MOCK_DUTCH_EXTRA = { auction_type:"dutch", clearing_price:5800, total_slots:10, slots_won:3, winners:[{id:"w1",name:"Ahmed K.",bid:6200,avatar:"A"},{id:"w2",name:"Sara M.",bid:6100,avatar:"S"},{id:"w3",name:"John D.",bid:5900,avatar:"J"}] };
const MOCK_REVERSE_EXTRA = { auction_type:"reverse", lowest_offer:38000, offers:[{id:"o1",vendor:"TechFix LLC",amount:38000},{id:"o2",vendor:"QuickServe Co.",amount:41500},{id:"o3",vendor:"ProBuild Ltd.",amount:44000},{id:"o4",vendor:"FastTools ME",amount:47800}] };

function AuctionTypeBadge({ auctionType }: { auctionType: string }) {
  const key = auctionType as keyof typeof AUCTION_TYPE_BADGE;
  const badge = AUCTION_TYPE_BADGE[key] ?? { label: auctionType, className: "bg-gray-100 text-gray-600" };
  return <span className={`text-[11px] font-bold px-2.5 py-1 rounded-full ${badge.className}`}>{badge.label}</span>;
}

function DutchAuctionPanel({ listing, auctionId, onBid, bidAmount, setBidAmount, bidMessage, setBidMessage, isPending, isAuthenticated, navigate }: any) {
  const extra = listing.auction_extra ?? MOCK_DUTCH_EXTRA;
  return (
    <div className="bg-amber-50 border border-amber-200 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-3"><Layers className="w-5 h-5 text-amber-600" /><span className="font-bold text-amber-800">Dutch Auction</span></div>
      <p className="text-sm text-amber-700 mb-3">Clearing price: <strong>{formatPrice(extra.clearing_price, listing.currency||"AED")}</strong> · {extra.slots_won}/{extra.total_slots} slots won</p>
      {extra.winners?.length>0 && <div className="flex gap-2 mb-3">{extra.winners.map((w:any)=><span key={w.id} className="bg-white border border-amber-200 rounded-full px-2.5 py-1 text-xs text-amber-800">{w.avatar} {w.name} — {formatPrice(w.bid,listing.currency||"AED")}</span>)}</div>}
      <div className="flex gap-2">
        <input type="number" value={bidAmount} onChange={e=>{setBidAmount(e.target.value);setBidMessage("");}} placeholder="Your bid" className="flex-1 border border-amber-300 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-amber-400" />
        <button onClick={()=>{if(!isAuthenticated){navigate("/login");return;}onBid();}} disabled={isPending} className="bg-amber-500 hover:bg-amber-600 text-white font-bold px-4 py-2 rounded-lg text-sm disabled:opacity-60">{isPending?"Bidding...":"Bid"}</button>
      </div>
      {bidMessage&&<p className={`text-xs mt-2 ${bidMessage.includes("success")?"text-green-700":"text-red-600"}`}>{bidMessage}</p>}
    </div>
  );
}

function ReverseAuctionPanel({ listing, onBid, bidAmount, setBidAmount, bidMessage, setBidMessage, isPending }: any) {
  const extra = listing.auction_extra ?? MOCK_REVERSE_EXTRA;
  return (
    <div className="bg-purple-50 border border-purple-200 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-3"><TrendingDown className="w-5 h-5 text-purple-600" /><span className="font-bold text-purple-800">Reverse Auction</span></div>
      <p className="text-sm text-purple-700 mb-3">Lowest offer: <strong>{formatPrice(extra.lowest_offer,listing.currency||"AED")}</strong></p>
      {extra.offers?.length>0 && <div className="space-y-1 mb-3">{extra.offers.map((o:any,i:number)=><div key={o.id} className={`flex justify-between text-xs px-2 py-1 rounded ${i===0?"bg-purple-100 text-purple-900 font-semibold":"text-purple-600"}`}><span>{o.vendor}</span><span>{formatPrice(o.amount,listing.currency||"AED")}</span></div>)}</div>}
      <div className="flex gap-2">
        <input type="number" value={bidAmount} onChange={e=>{setBidAmount(e.target.value);setBidMessage("");}} placeholder="Your offer (lower wins)" className="flex-1 border border-purple-300 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-purple-400" />
        <button onClick={()=>onBid(extra.lowest_offer)} disabled={isPending} className="bg-purple-500 hover:bg-purple-600 text-white font-bold px-4 py-2 rounded-lg text-sm disabled:opacity-60">{isPending?"Submitting...":"Submit Offer"}</button>
      </div>
      {bidMessage&&<p className={`text-xs mt-2 ${bidMessage.includes("success")?"text-green-700":"text-red-600"}`}>{bidMessage}</p>}
    </div>
  );
}

function StarRating({ rating, count }: { rating: number; count?: number }) {
  return (
    <div className="flex items-center gap-1">
      {[1,2,3,4,5].map(s=><Star key={s} size={14} fill={s<=Math.round(rating)?"#FFC220":"none"} className={s<=Math.round(rating)?"text-[#FFC220]":"text-gray-300"} />)}
      {count!=null&&<span className="text-xs text-gray-500 ml-1">({count})</span>}
    </div>
  );
}

function FeedbackDist({ dist, total }: { dist: Record<number,number>; total: number }) {
  return (
    <div className="space-y-1">
      {[5,4,3,2,1].map(star=>{const c=dist[star]||0;const pct=total>0?(c/total)*100:0;return(
        <div key={star} className="flex items-center gap-2 text-xs">
          <span className="w-3 text-right">{star}</span><Star size={10} fill="#FFC220" className="text-[#FFC220]" />
          <div className="flex-1 h-2 bg-gray-100 rounded-full overflow-hidden"><div className="h-full bg-[#FFC220] rounded-full" style={{width:`${pct}%`}} /></div>
          <span className="w-6 text-gray-500">{c}</span>
        </div>
      );})}
    </div>
  );
}

const CONDITION_LABELS: Record<string,string> = { new:"New", "like-new":"Open Box — Like New", good:"Used — Good", fair:"Used — Fair", "for-parts":"For parts or not working", refurbished:"Manufacturer Refurbished" };

export default function ListingDetailPage() {
  const params = (useParams() as { id: string });
  const id = params.id;
  const router = useRouter();
  const navigate = (path: string) => router.push(path);
  const { isAuthenticated } = useAuthStore();
  const qc = useQueryClient();
  const [activeImage, setActiveImage] = useState(0);
  const [bidAmount, setBidAmount] = useState("");
  const [bidMessage, setBidMessage] = useState("");
  const [cartMessage, setCartMessage] = useState("");
  const [watchlistMessage, setWatchlistMessage] = useState("");
  const [chatOpen, setChatOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<"description"|"shipping"|"qa"|"feedback">("description");
  const [selectedVariant, setSelectedVariant] = useState<any>(null);
  const [zoomPos, setZoomPos] = useState<{x:number;y:number}|null>(null);
  const [questionText, setQuestionText] = useState("");
  const imgRef = useRef<HTMLDivElement>(null);

  const { data: apiListing, isLoading } = useQuery({ queryKey:["listing",id], queryFn:()=>api.get(`/listings/${id}`,{timeout:4000}).then(r=>r.data.data), retry:0, staleTime:30_000 });
  const { data: variantsData=[] } = useQuery({ queryKey:["listing-variants",id], queryFn:()=>api.get(`/listings/${id}/variants`).then(r=>r.data?.data??r.data).catch(()=>[]), retry:0 });
  const { data: qaData=[] } = useQuery({ queryKey:["listing-qa",id], queryFn:()=>api.get(`/listings/${id}/questions`).then(r=>r.data?.data??r.data).catch(()=>[]), retry:0 });
  const { data: feedbackData } = useQuery({ queryKey:["listing-feedback",id], queryFn:()=>api.get(`/listings/${id}/feedback`).then(r=>r.data?.data??r.data).catch(()=>null), retry:0 });
  const { data: recentlyViewed=[] } = useQuery({ queryKey:["recently-viewed"], queryFn:()=>api.get("/listings/recently-viewed").then(r=>r.data?.data??r.data).catch(()=>[]), retry:0, enabled:isAuthenticated });

  const variants: any[] = Array.isArray(variantsData) ? variantsData : [];
  const questions: any[] = Array.isArray(qaData) ? qaData : [];
  const feedback = feedbackData as any;
  const feedbackList: any[] = feedback?.feedback ?? [];
  const feedbackSummary = feedback?.summary;

  useEffect(()=>{ if(id&&apiListing) api.post(`/listings/${id}/view`).catch(()=>{}); },[id,apiListing]);

  const mockFallbackListing = (()=>{
    if(apiListing) return null;
    const m = ALL_LISTINGS.find(l=>l.id===id); if(!m) return null;
    return { id:m.id, title:m.title, description:`Premium ${m.category.toLowerCase()} listing available in ${m.location}. In ${m.condition.toLowerCase()} condition.`,
      price:m.price, currency:m.currency, category:{slug:m.category.toLowerCase().replace(/\s+/g,"-"),name:m.category},
      location:{city:m.location.split(",")[0].trim(),country:"UAE"}, condition:m.condition, images:[{url:m.image}],
      type:"fixed", is_auction:false, seller:{id:"seller_demo",name:m.seller,rating:m.rating,verified:true},
      attributes:{}, created_at:m.created_at, views:247, saves:38 };
  })();

  const listing = apiListing || mockFallbackListing;
  const bidMutation = useMutation({ mutationFn:(amount:number)=>api.post(`/listings/${id}/bid`,{amount}).then(r=>r.data), onSuccess:()=>{setBidMessage("Bid placed successfully! 🎉");qc.invalidateQueries({queryKey:["listing",id]});}, onError:(err:any)=>{setBidMessage(err?.response?.data?.message||"Failed to place bid.");} });
  const watchlistMutation = useMutation({ mutationFn:async(next:boolean)=>{if(next){await addWatchlistItem(id);if(listing?.price&&listing.price>0)setWatchlistPriceSnapshot(id,listing.price);}else{await removeWatchlistItem(id);removeWatchlistPriceSnapshot(id);}return next;}, onSuccess:()=>{qc.invalidateQueries({queryKey:["listing",id]});qc.invalidateQueries({queryKey:["watchlist"]});}, onError:(err:any)=>{setWatchlistMessage(err?.response?.data?.message||"Could not update watchlist.");} });
  const addToCartMutation = useMutation({ mutationFn:()=>addCartItem(id,1), onSuccess:()=>{setCartMessage("Added to cart successfully.");qc.invalidateQueries({queryKey:["cart"]});}, onError:(err:any)=>{setCartMessage(err?.response?.data?.message||"Failed to add item to cart.");} });
  const askQuestionMut = useMutation({ mutationFn:(q:string)=>api.post(`/listings/${id}/questions`,{question:q}).then(r=>r.data), onSuccess:()=>{qc.invalidateQueries({queryKey:["listing-qa",id]});setQuestionText("");} });

  const handleBid = (currentLowestOffer?:number)=>{
    if(!isAuthenticated){router.push("/login");return;}
    const amount=Number(bidAmount); if(!amount||amount<=0){setBidMessage("Please enter a valid amount.");return;}
    if(auctionType==="reverse"&&currentLowestOffer!=null&&amount>=currentLowestOffer){setBidMessage(`Your offer must be lower than the current lowest (${formatPrice(currentLowestOffer,listing?.currency||"AED")}).`);return;}
    bidMutation.mutate(amount);
  };

  const handleMouseMove = useCallback((e:React.MouseEvent<HTMLDivElement>)=>{
    if(!imgRef.current) return;
    const rect=imgRef.current.getBoundingClientRect();
    setZoomPos({x:((e.clientX-rect.left)/rect.width)*100, y:((e.clientY-rect.top)/rect.height)*100});
  },[]);

  if(isLoading) return (<div className="max-w-7xl mx-auto px-4 py-10"><div className="grid grid-cols-1 lg:grid-cols-[1fr_380px] gap-8"><div className="grid grid-cols-[72px_1fr] gap-3"><div className="space-y-2">{Array.from({length:5}).map((_,i)=><div key={i} className="w-full aspect-square bg-gray-100 rounded animate-pulse" />)}</div><div className="h-[500px] bg-gray-100 rounded-xl animate-pulse" /></div><div className="space-y-4"><div className="h-6 bg-gray-100 rounded animate-pulse w-1/3" /><div className="h-8 bg-gray-100 rounded animate-pulse" /><div className="h-10 bg-gray-100 rounded animate-pulse w-1/2" /><div className="h-40 bg-gray-100 rounded animate-pulse" /></div></div></div>);
  if(!listing) return (<div className="text-center py-20 text-gray-400"><p className="text-5xl mb-4">😕</p><p className="text-lg font-semibold">Listing not found</p><button onClick={()=>router.push("/listings")} className="mt-4 text-[#0071CE] hover:underline text-sm">← Back to listings</button></div>);

  const images = listing.images?.length ? listing.images.map((img:any)=>img.url||img) : [`https://picsum.photos/seed/${id}/600/400`];
  const isAuction = listing.type==="auction"||listing.is_auction;
  const auctionType = getAuctionType(listing);
  const price = isAuction ? listing.current_bid??listing.start_price??0 : listing.price??0;
  const categorySlug = listing.category?.slug;
  const schema = getCategorySchema(categorySlug);
  const attributes: Record<string,unknown> = listing.attributes??{};
  const specRows = schema ? schema.fields.map(f=>{const raw=attributes[f.name];const fmt=formatFieldValue(f,raw);if(!fmt)return null;return{label:f.label,value:fmt};}).filter((r):r is {label:string;value:string}=>r!==null) : [];
  const priceLabel = auctionType==="dutch"?"Clearing Price":auctionType==="reverse"?"Lowest Offer":"Current Bid";
  const displayPrice = auctionType==="dutch"?(listing.clearing_price??price):auctionType==="reverse"?(listing.lowest_offer??price):price;
  const sellerLabel = auctionType==="reverse"?"Buyer":"Seller";
  const isExpired = Boolean(listing.expires_at&&new Date(listing.expires_at).getTime()<Date.now());
  const isSoldOrUnavailable = ["sold","expired","inactive"].includes((listing.status||"").toLowerCase())||isExpired;
  const isWatched = listing.is_watched??listing.isWatched??false;
  const effectivePrice = selectedVariant?selectedVariant.price:price;
  const effectiveCompareAt = selectedVariant?.compare_at_price??null;

  let cfObj: Record<string,string>={};
  if(listing.custom_fields){try{cfObj=typeof listing.custom_fields==="string"?JSON.parse(listing.custom_fields):listing.custom_fields;}catch{}}
  const cfEntries = Object.entries(cfObj).filter(([,v])=>v);

  const variantAttrGroups: Record<string,string[]>={};
  if(variants.length>0){variants.forEach((v:any)=>{try{const a=typeof v.attributes==="string"?JSON.parse(v.attributes):v.attributes||{};Object.entries(a).forEach(([k,val])=>{if(!variantAttrGroups[k])variantAttrGroups[k]=[];if(!variantAttrGroups[k].includes(String(val)))variantAttrGroups[k].push(String(val));});}catch{}});}

  return (
    <div className="max-w-7xl mx-auto px-4 py-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-1.5 text-xs text-gray-500 mb-4 overflow-x-auto">
        <Link href="/" className="hover:text-[#0071CE]">Home</Link><ChevronRight className="w-3 h-3 text-gray-300 shrink-0" />
        {listing.category?.name&&<><Link href={`/category/${listing.category.slug||""}`} className="hover:text-[#0071CE]">{listing.category.name}</Link><ChevronRight className="w-3 h-3 text-gray-300 shrink-0" /></>}
        <span className="text-gray-800 font-medium truncate">{listing.title}</span>
      </nav>

      {/* Main: Image + Buy Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_380px] gap-8">
        {/* LEFT: Image Gallery with zoom */}
        <div className="grid grid-cols-[72px_1fr] gap-3">
          <div className="space-y-2">
            {images.map((img:string,i:number)=>(
              <button key={i} onMouseEnter={()=>setActiveImage(i)} className={`w-full aspect-square rounded-lg overflow-hidden border-2 transition-all ${activeImage===i?"border-[#0071CE] shadow-sm":"border-gray-200 hover:border-gray-400"}`}>
                <img src={img} alt={`thumb-${i}`} className="w-full h-full object-cover" />
              </button>
            ))}
          </div>
          <div ref={imgRef} onMouseEnter={()=>setZoomPos({x:50,y:50})} onMouseMove={handleMouseMove} onMouseLeave={()=>setZoomPos(null)} className="relative rounded-xl overflow-hidden bg-gray-50 cursor-crosshair border border-gray-200">
            <img src={images[activeImage]} alt={listing.title} className="w-full h-[500px] object-contain" onError={e=>{(e.target as HTMLImageElement).src=`https://picsum.photos/seed/${id}/600/400`;}} />
            {zoomPos&&<div className="absolute inset-0 pointer-events-none" style={{backgroundImage:`url(${images[activeImage]})`,backgroundSize:"200%",backgroundPosition:`${zoomPos.x}% ${zoomPos.y}%`,backgroundRepeat:"no-repeat"}} />}
            <div className="absolute bottom-3 right-3 bg-white/80 rounded-lg px-2 py-1 flex items-center gap-1 text-xs text-gray-500"><ZoomIn className="w-3 h-3" /> Hover to zoom</div>
            {images.length>1&&<div className="absolute bottom-3 left-3 flex gap-1">
              <button onClick={()=>setActiveImage(Math.max(0,activeImage-1))} className="w-8 h-8 bg-white/80 rounded-full flex items-center justify-center hover:bg-white"><ChevronLeft className="w-4 h-4" /></button>
              <button onClick={()=>setActiveImage(Math.min(images.length-1,activeImage+1))} className="w-8 h-8 bg-white/80 rounded-full flex items-center justify-center hover:bg-white"><ChevronRight className="w-4 h-4" /></button>
            </div>}
          </div>
        </div>

        {/* RIGHT: Buy Panel */}
        <div>
          {/* Title */}
          <div className="bg-white border border-gray-200 rounded-t-xl p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1.5">{isAuction&&<AuctionTypeBadge auctionType={auctionType} />}</div>
                <h1 className="text-lg font-bold text-gray-900 leading-snug">{listing.title}</h1>
              </div>
              <div className="flex gap-1.5 shrink-0">
                <button onClick={()=>{if(!isAuthenticated){router.push("/login?next=/listings/"+id);return;}setWatchlistMessage("");watchlistMutation.mutate(!isWatched,{onSuccess:()=>setWatchlistMessage(!isWatched?"Added to watchlist.":"Removed from watchlist.")});}} disabled={watchlistMutation.isPending} className={`p-2 border rounded-lg transition-colors disabled:opacity-60 ${isWatched?"bg-red-500 border-red-500 text-white":"hover:bg-gray-50"}`} title="Watch this item">
                  <Heart size={16} className={isWatched?"text-white":"text-gray-400"} fill={isWatched?"currentColor":"none"} />
                </button>
                <button className="p-2 border rounded-lg hover:bg-gray-50 transition-colors" title="Share"><Share2 size={16} className="text-gray-400" /></button>
              </div>
            </div>
            {watchlistMessage&&<p className={`text-xs mt-1 ${watchlistMessage.includes("Added")?"text-green-600":"text-red-500"}`}>{watchlistMessage}</p>}
          </div>

          {/* Price */}
          <div className="bg-white border-x border-gray-200 px-4 py-3">
            {isAuction?(
              <div>
                <p className="text-xs text-gray-500 mb-0.5">{priceLabel}</p>
                <p className="text-3xl font-extrabold text-[#0071CE]">{formatPrice(displayPrice,listing.currency||"AED")}</p>
                {listing.ends_at&&<div className="mt-2 flex items-center gap-2 text-sm"><Clock className="w-4 h-4 text-red-500" /><span className="text-gray-600">Ends in</span><CountdownTimer endsAt={listing.ends_at} /></div>}
              </div>
            ):(
              <div>
                <div className="flex items-baseline gap-3">
                  <p className="text-3xl font-extrabold text-gray-900">{formatPrice(effectivePrice,listing.currency||"AED")}</p>
                  {effectiveCompareAt>0&&<p className="text-lg text-gray-400 line-through">{formatPrice(effectiveCompareAt,listing.currency||"AED")}</p>}
                </div>
                {listing.price_type==="negotiable"&&<p className="text-xs text-orange-600 font-medium mt-1">Price negotiable — make an offer</p>}
              </div>
            )}
          </div>

          {/* Condition */}
          <div className="bg-white border-x border-gray-200 px-4 py-2.5">
            <div className="flex items-center gap-2">
              <span className={`text-sm font-semibold ${listing.condition==="new"?"text-green-700":"text-gray-700"}`}>{CONDITION_LABELS[listing.condition]||listing.condition||"Not specified"}</span>
              {listing.condition==="new"&&<Shield className="w-4 h-4 text-green-600" />}
            </div>
          </div>

          {/* Variants */}
          {variants.length>0&&(
            <div className="bg-white border-x border-gray-200 px-4 py-3 space-y-3">
              {Object.entries(variantAttrGroups).map(([attrName,values])=>(
                <div key={attrName}>
                  <p className="text-xs font-semibold text-gray-500 uppercase mb-1.5">{attrName}</p>
                  <div className="flex flex-wrap gap-1.5">
                    {values.map(val=>{const mv=variants.find((v:any)=>{try{const a=typeof v.attributes==="string"?JSON.parse(v.attributes):v.attributes;return a[attrName]===val;}catch{return false;}});const sel=selectedVariant?.id===mv?.id;return(
                      <button key={val} onClick={()=>setSelectedVariant(mv||null)} className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${sel?"border-[#0071CE] bg-blue-50 text-[#0071CE] font-semibold":"border-gray-200 hover:border-gray-400"}`}>{val}</button>
                    );})}
                  </div>
                </div>
              ))}
              {selectedVariant&&selectedVariant.stock>=0&&<p className={`text-xs ${selectedVariant.stock>0?"text-green-600":"text-red-500"}`}>{selectedVariant.stock>0?`${selectedVariant.stock} in stock`:"Out of stock"}</p>}
            </div>
          )}

          {/* Delivery info */}
          <div className="bg-white border-x border-gray-200 px-4 py-3 space-y-2">
            <div className="flex items-center gap-2 text-sm"><Truck className="w-4 h-4 text-gray-500" /><span className="text-gray-700">Standard delivery</span><span className="text-gray-500">— estimated 3-5 business days</span></div>
            <div className="flex items-center gap-2 text-sm"><RotateCcw className="w-4 h-4 text-gray-500" /><span className="text-gray-700">Returns accepted</span><span className="text-gray-500">— within 14 days</span></div>
          </div>

          {/* Conversion signals / Urgency */}
          {FeatureFlags.conversionSignals&&<div className="bg-white border-x border-gray-200 px-4 py-2"><ConversionSignals watchersCount={listing.watchers_count??listing.saves} viewsToday={listing.views_today??listing.views} offersCount={listing.offers_count} bidCount={listing.bid_count??listing.bids_count} /></div>}
          {FeatureFlags.urgencyBadges&&<div className="bg-white border-x border-gray-200 px-4 py-2"><UrgencyBadge watchersCount={listing.watchers_count??listing.saves} viewsToday={listing.views_today??listing.views} offersCount={listing.offers_count} endsAt={listing.ends_at} isAuction={isAuction} /></div>}

          {/* Action buttons */}
          <div className="bg-white border border-gray-200 rounded-b-xl p-4 space-y-2.5">
            {isAuction?(
              auctionType==="dutch"?<DutchAuctionPanel listing={listing} auctionId={listing.auction_id||id} onBid={handleBid} bidAmount={bidAmount} setBidAmount={setBidAmount} bidMessage={bidMessage} setBidMessage={setBidMessage} isPending={bidMutation.isPending} isAuthenticated={isAuthenticated} navigate={navigate} />
              :auctionType==="reverse"?<ReverseAuctionPanel listing={listing} onBid={handleBid} bidAmount={bidAmount} setBidAmount={setBidAmount} bidMessage={bidMessage} setBidMessage={setBidMessage} isPending={bidMutation.isPending} />
              :(<>
                <div className="flex gap-2">
                  <input type="number" value={bidAmount} onChange={e=>{setBidAmount(e.target.value);setBidMessage("");}} placeholder={`Min: ${formatPrice(Number(price)+100,listing.currency||"AED")}`} className="flex-1 border border-gray-200 rounded-lg px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]" />
                  <button onClick={()=>handleBid()} disabled={bidMutation.isPending} className="bg-red-500 hover:bg-red-600 text-white font-bold px-4 py-2.5 rounded-lg text-sm disabled:opacity-60">{bidMutation.isPending?"Bidding...":"🔨 Bid"}</button>
                </div>
                {bidMessage&&<p className={`text-xs ${bidMessage.includes("success")?"text-green-600":"text-red-500"}`}>{bidMessage}</p>}
              </>)
            ):(
              <>
                <button onClick={()=>{if(!isAuthenticated){router.push("/login?next=/listings/"+id);return;}setCartMessage("");addToCartMutation.mutate(undefined,{onSuccess:()=>router.push("/cart")});}} disabled={addToCartMutation.isPending||isSoldOrUnavailable} className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-lg transition-colors text-sm disabled:opacity-50">{addToCartMutation.isPending?"Processing...":`Buy It Now · ${formatPrice(effectivePrice,listing.currency||"AED")}`}</button>
                <button onClick={()=>{if(!isAuthenticated){router.push("/login?next=/listings/"+id);return;}setCartMessage("");addToCartMutation.mutate();}} disabled={addToCartMutation.isPending||isSoldOrUnavailable} className="w-full border-2 border-[#0071CE] text-[#0071CE] font-bold py-3 rounded-lg hover:bg-blue-50 transition-colors text-sm disabled:opacity-50">{addToCartMutation.isPending?"Adding...":"Add to Cart"}</button>
                {isSoldOrUnavailable&&<p className="text-xs text-red-500">This listing is sold out or expired.</p>}
                {cartMessage&&<p className={`text-xs ${cartMessage.includes("success")||cartMessage.includes("Added")?"text-green-600":"text-red-500"}`}>{cartMessage}</p>}
              </>
            )}
            <button onClick={()=>{if(!isAuthenticated){router.push("/login");return;}setChatOpen(true);}} className="w-full border border-gray-300 text-gray-700 font-semibold py-2.5 rounded-lg hover:bg-gray-50 transition-colors text-sm flex items-center justify-center gap-2"><MessageCircle size={16} /> Message {sellerLabel}</button>
            {FeatureFlags.offerActions&&!isAuction&&!isSoldOrUnavailable&&<OfferActions listingId={id} listingPrice={effectivePrice} currency={listing.currency||"AED"} isAuction={isAuction} />}
          </div>

          {/* Seller card */}
          {listing.seller&&(
            <div className="bg-white border border-gray-200 rounded-xl p-4 mt-3">
              <div className="flex items-center gap-3">
                <Link href={`/sellers/${listing.seller.id}`} className="w-11 h-11 rounded-full bg-[#0071CE] flex items-center justify-center text-white font-bold text-lg shrink-0">{listing.seller.name?.[0]||"S"}</Link>
                <div className="flex-1 min-w-0">
                  <Link href={`/sellers/${listing.seller.id}`} className="font-semibold text-sm text-gray-800 hover:text-[#0071CE] transition-colors truncate block">{listing.seller.name}</Link>
                  <div className="flex items-center gap-2 mt-0.5">
                    {listing.seller.rating&&<StarRating rating={listing.seller.rating} />}
                    <span className="text-xs text-gray-400">{sellerLabel}</span>
                    {listing.seller.verified&&<span className="flex items-center gap-0.5 text-xs text-green-600 font-medium"><Shield className="w-3 h-3" /> Verified</span>}
                  </div>
                </div>
              </div>
              <Link href={`/sellers/${listing.seller.id}`} className="mt-3 w-full flex items-center justify-center gap-1.5 border border-gray-200 rounded-lg py-2 text-xs text-gray-600 hover:bg-white hover:border-[#0071CE] hover:text-[#0071CE] transition-colors"><Store size={13} /> View Storefront</Link>
              {FeatureFlags.trustBadges&&<div className="mt-2"><TrustBadges isVerifiedSeller={listing.seller.verified} rating={listing.seller.rating} trustLevel={listing.seller.trust_level??(listing.seller.rating!=null&&listing.seller.rating>=4?"high":listing.seller.rating>=3?"medium":undefined)} /></div>}
            </div>
          )}

          {FeatureFlags.crowdshippingWidget&&!isAuction&&<CrowdshippingWidget listingId={id} sellerCity={listing.city} currency={listing.currency||"AED"} />}
          {FeatureFlags.priceBreakdown&&!isAuction&&effectivePrice>0&&<PriceBreakdown itemPrice={effectivePrice} platformFee={listing.platform_fee} total={listing.total_price??effectivePrice} currency={listing.currency||"AED"} escrowed />}
          {FeatureFlags.inlineHints&&!isAuction&&<div className="mt-2">{listing.saves!=null&&listing.saves>10&&<InlineHint type="high_demand" message="High demand item — act fast!" />}{!isAuction&&effectivePrice>0&&<InlineHint type="price_tip" message="Try lowering your price in the offer to negotiate" />}</div>}
        </div>
      </div>

      {/* Tabs Section */}
      <div className="mt-8 border border-gray-200 rounded-xl overflow-hidden bg-white">
        <div className="flex border-b border-gray-200 bg-gray-50">
          {[{key:"description",label:"Description",icon:null},{key:"shipping",label:"Shipping & Returns",icon:Truck},{key:"qa",label:`Q&A (${questions.length})`,icon:HelpCircle},{key:"feedback",label:"Feedback",icon:Star}].map(tab=>(
            <button key={tab.key} onClick={()=>setActiveTab(tab.key as any)} className={`flex items-center gap-1.5 px-5 py-3 text-sm font-semibold transition-colors border-b-2 ${activeTab===tab.key?"border-[#0071CE] text-[#0071CE] bg-white":"border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100"}`}>
              {tab.icon&&<tab.icon className="w-4 h-4" />}{tab.label}
            </button>
          ))}
        </div>
        <div className="p-6">
          {activeTab==="description"&&(
            <div>
              <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">{listing.description||"No description provided."}</p>
              {(specRows.length>0||cfEntries.length>0)&&(
                <div className="mt-6 border border-gray-200 rounded-lg overflow-hidden">
                  <table className="w-full text-sm"><tbody>
                    {specRows.map((row,i)=><tr key={row.label} className={i%2===0?"bg-gray-50":"bg-white"}><td className="px-4 py-2.5 font-semibold text-gray-600 w-1/3">{row.label}</td><td className="px-4 py-2.5 text-gray-900">{row.value}</td></tr>)}
                    {cfEntries.map(([key,val],i)=><tr key={key} className={(specRows.length+i)%2===0?"bg-gray-50":"bg-white"}><td className="px-4 py-2.5 font-semibold text-gray-600 w-1/3">{key.replace(/_/g," ")}</td><td className="px-4 py-2.5 text-gray-900">{val}</td></tr>)}
                  </tbody></table>
                </div>
              )}
            </div>
          )}
          {activeTab==="shipping"&&(
            <div className="space-y-4">
              <div className="flex items-start gap-3"><Truck className="w-5 h-5 text-gray-500 mt-0.5 shrink-0" /><div><p className="font-semibold text-gray-800">Standard Delivery</p><p className="text-sm text-gray-600">Estimated 3-5 business days</p><p className="text-sm text-gray-500 mt-1">Seller ships from: {listing.city||listing.country||"Not specified"}</p></div></div>
              <div className="flex items-start gap-3"><RotateCcw className="w-5 h-5 text-gray-500 mt-0.5 shrink-0" /><div><p className="font-semibold text-gray-800">Return Policy</p><p className="text-sm text-gray-600">Returns accepted within 14 days of delivery</p><p className="text-sm text-gray-500 mt-1">Item must be in original condition. Buyer pays return shipping.</p></div></div>
              <div className="flex items-start gap-3"><Shield className="w-5 h-5 text-gray-500 mt-0.5 shrink-0" /><div><p className="font-semibold text-gray-800">Buyer Protection</p><p className="text-sm text-gray-600">Full refund if item not delivered or not as described</p><p className="text-sm text-gray-500 mt-1">Escrow payment held until delivery confirmed.</p></div></div>
            </div>
          )}
          {activeTab==="qa"&&(
            <div className="space-y-4">
              {isAuthenticated?(
                <div className="flex gap-2">
                  <input value={questionText} onChange={e=>setQuestionText(e.target.value)} placeholder="Ask the seller a question..." className="flex-1 border border-gray-200 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]" />
                  <button onClick={()=>{if(questionText.trim().length>=5)askQuestionMut.mutate(questionText);}} disabled={askQuestionMut.isPending||questionText.trim().length<5} className="bg-[#0071CE] text-white font-semibold px-4 py-2 rounded-lg text-sm hover:bg-[#005BA1] disabled:opacity-50 flex items-center gap-1.5"><Send className="w-3.5 h-3.5" /> Ask</button>
                </div>
              ):<p className="text-sm text-gray-500"><Link href="/login" className="text-[#0071CE] hover:underline">Sign in</Link> to ask a question</p>}
              {questions.length===0&&<p className="text-sm text-gray-400 text-center py-6">No questions yet. Be the first to ask!</p>}
              {questions.map((q:any)=>(
                <div key={q.id} className="border border-gray-100 rounded-lg p-4">
                  <div className="flex items-start gap-2"><HelpCircle className="w-4 h-4 text-gray-400 mt-0.5 shrink-0" /><div className="flex-1">
                    <p className="text-sm text-gray-800">{q.question}</p>
                    <p className="text-xs text-gray-400 mt-1">{new Date(q.created_at).toLocaleDateString()}</p>
                    {q.answer?(<div className="mt-2 ml-4 pl-3 border-l-2 border-[#0071CE]/30"><p className="text-sm text-gray-700">{q.answer}</p><p className="text-xs text-gray-400 mt-1">Answered {q.answered_at?new Date(q.answered_at).toLocaleDateString():""}</p><div className="flex items-center gap-3 mt-1.5"><button className="text-xs text-gray-400 hover:text-green-600 flex items-center gap-0.5"><ThumbsUp className="w-3 h-3" /> Helpful ({q.helpful_yes||0})</button></div></div>)
                    :<p className="text-xs text-amber-600 mt-2 ml-4 pl-3 border-l-2 border-amber-200">Awaiting seller response</p>}
                  </div></div>
                </div>
              ))}
            </div>
          )}
          {activeTab==="feedback"&&(
            <div className="space-y-4">
              {feedbackSummary&&feedbackSummary.count>0?(
                <>
                  <div className="flex gap-6 items-start">
                    <div className="text-center"><p className="text-4xl font-bold text-gray-900">{feedbackSummary.avg_rating?.toFixed(1)}</p><StarRating rating={feedbackSummary.avg_rating} count={feedbackSummary.count} /><p className="text-xs text-gray-500 mt-1">{feedbackSummary.count} reviews</p></div>
                    <div className="flex-1"><FeedbackDist dist={feedbackSummary.distribution} total={feedbackSummary.count} /></div>
                  </div>
                  <div className="divide-y divide-gray-100">
                    {feedbackList.map((fb:any)=>(
                      <div key={fb.id} className="py-4">
                        <div className="flex items-center gap-2"><StarRating rating={fb.rating} />{fb.title&&<span className="text-sm font-semibold text-gray-800">{fb.title}</span>}</div>
                        {fb.review&&<p className="text-sm text-gray-600 mt-1.5">{fb.review}</p>}
                        <p className="text-xs text-gray-400 mt-1">{fb.is_anonymous?"Anonymous":fb.buyer_id?.slice(0,8)} · {new Date(fb.created_at).toLocaleDateString()}</p>
                      </div>
                    ))}
                  </div>
                </>
              ):<p className="text-sm text-gray-400 text-center py-6">No feedback yet for this listing.</p>}
            </div>
          )}
        </div>
      </div>

      {/* Similar Listings */}
      {(()=>{
        const cat=ALL_LISTINGS.find(l=>l.id===id);
        const catStr=typeof listing.category==="string"?listing.category:listing.category?.name||"Other";
        const locStr=typeof listing.location==="string"?listing.location:listing.location?.city?`${listing.location.city}, UAE`:"Dubai, UAE";
        const forRec=cat||{id:id||"unknown",title:listing.title||"Listing",price:typeof price==="number"?price:Number(price)||0,currency:listing.currency||"AED",category:catStr,location:locStr,condition:listing.condition||"Good",image:images[0]||`https://picsum.photos/seed/${id}/400/300`,seller:listing.seller?.name||"Seller",rating:listing.seller?.rating||4.5,created_at:listing.created_at||new Date().toISOString()};
        return <SimilarListings listing={forRec} />;
      })()}

      {/* Recently Viewed */}
      {recentlyViewed.length>0&&Array.isArray(recentlyViewed)&&recentlyViewed.length>0&&(
        <div className="mt-8">
          <h2 className="text-lg font-bold text-gray-900 mb-4">Recently Viewed</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-6 gap-3">
            {recentlyViewed.slice(0,6).map((item:any)=>(
              <button key={item.id} onClick={()=>router.push(`/listings/${item.id}`)} className="bg-white rounded-xl overflow-hidden border border-gray-100 hover:shadow-md hover:border-[#0071CE]/30 transition-all text-left">
                <img src={item.images?.[0]?.url||`https://picsum.photos/seed/${item.id}/200/200`} alt={item.title} className="w-full h-24 object-cover" />
                <div className="p-2"><p className="text-xs font-semibold text-gray-800 line-clamp-2">{item.title}</p><p className="text-xs text-[#0071CE] font-bold mt-1">{item.currency||"AED"} {item.price?.toLocaleString()}</p></div>
              </button>
            ))}
          </div>
        </div>
      )}

      {chatOpen&&listing.seller&&<ChatPanel sellerId={listing.seller.id} sellerName={listing.seller.name||"Seller"} listingId={id} onClose={()=>setChatOpen(false)} />}
    </div>
  );
}
