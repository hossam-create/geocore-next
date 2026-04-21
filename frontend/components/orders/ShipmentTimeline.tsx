'use client';

import { CheckCircle2, Circle, Truck, Package, MapPin, Home, Clock, XCircle, Upload } from 'lucide-react';

export type ShipmentStatus =
  | 'requested'
  | 'accepted'
  | 'purchased'
  | 'in_transit'
  | 'arrived_country'
  | 'out_for_delivery'
  | 'delivered'
  | 'confirmed'
  | 'cancelled';

interface TrackingEvent {
  status: ShipmentStatus;
  location?: string;
  note?: string;
  proof_image_url?: string;
  created_at: string;
}

interface ShipmentTimelineProps {
  events: TrackingEvent[];
  currentStatus?: ShipmentStatus;
  onUploadProof?: () => void;
  onConfirmDelivery?: () => void;
  isBuyer?: boolean;
}

const STATUS_META: Record<ShipmentStatus, { label: string; icon: typeof Truck; color: string }> = {
  requested: { label: 'Shipment Requested', icon: Package, color: 'text-gray-500' },
  accepted: { label: 'Traveler Accepted', icon: CheckCircle2, color: 'text-blue-600' },
  purchased: { label: 'Item Purchased', icon: Package, color: 'text-blue-600' },
  in_transit: { label: 'In Transit', icon: Truck, color: 'text-purple-600' },
  arrived_country: { label: 'Arrived in Country', icon: MapPin, color: 'text-purple-600' },
  out_for_delivery: { label: 'Out for Delivery', icon: Home, color: 'text-amber-600' },
  delivered: { label: 'Delivered', icon: CheckCircle2, color: 'text-green-600' },
  confirmed: { label: 'Delivery Confirmed', icon: CheckCircle2, color: 'text-green-700' },
  cancelled: { label: 'Cancelled', icon: XCircle, color: 'text-red-600' },
};

const FULL_CHAIN: ShipmentStatus[] = [
  'requested', 'accepted', 'purchased', 'in_transit',
  'arrived_country', 'out_for_delivery', 'delivered', 'confirmed',
];

export function ShipmentTimeline({ events, currentStatus, onUploadProof, onConfirmDelivery, isBuyer }: ShipmentTimelineProps) {
  const eventMap = new Map(events.map(e => [e.status, e]));
  const reachedStatus = currentStatus || (events.length > 0 ? events[events.length - 1].status : null);
  const reachedIndex = reachedStatus ? FULL_CHAIN.indexOf(reachedStatus) : -1;
  const isCancelled = currentStatus === 'cancelled' || reachedStatus === 'cancelled';

  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5">
      <h3 className="mb-4 text-sm font-bold text-gray-800 flex items-center gap-2">
        <Truck size={16} className="text-[#0071CE]" /> Shipment Tracking
      </h3>
      <ol className="space-y-3">
        {FULL_CHAIN.map((status, i) => {
          const event = eventMap.get(status);
          const done = i <= reachedIndex;
          const current = i === reachedIndex;
          const meta = STATUS_META[status];
          const Icon = meta.icon;

          return (
            <li key={status} className={`flex items-start gap-3 ${!done && !current ? 'opacity-40' : ''}`}>
              <span className="mt-0.5">
                {done ? (
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                ) : current ? (
                  <Icon className={`h-4 w-4 ${meta.color} animate-pulse`} />
                ) : (
                  <Circle className="h-4 w-4 text-gray-300" />
                )}
              </span>
              <div className="flex-1">
                <p className={`text-sm font-medium ${done || current ? 'text-gray-900' : 'text-gray-500'}`}>
                  {meta.label}
                </p>
                {event && (
                  <div className="text-xs text-gray-400 space-y-0.5">
                    {event.created_at && <p>{new Date(event.created_at).toLocaleString()}</p>}
                    {event.location && <p className="flex items-center gap-1"><MapPin size={10} /> {event.location}</p>}
                    {event.note && <p>{event.note}</p>}
                    {event.proof_image_url && (
                      <a href={event.proof_image_url} target="_blank" rel="noopener noreferrer" className="text-[#0071CE] hover:underline flex items-center gap-1">
                        <Upload size={10} /> View proof
                      </a>
                    )}
                  </div>
                )}
              </div>
            </li>
          );
        })}

        {isCancelled && (
          <li className="flex items-start gap-3">
            <XCircle className="mt-0.5 h-4 w-4 text-red-600" />
            <div>
              <p className="text-sm font-medium text-red-700">Shipment Cancelled</p>
            </div>
          </li>
        )}
      </ol>

      {/* Action buttons — prominent CTAs */}
      <div className="mt-5 space-y-3">
        {isBuyer && reachedStatus === 'delivered' && (
          <div className="rounded-xl border-2 border-green-300 bg-green-50 p-4 text-center space-y-2">
            <p className="text-sm font-semibold text-green-800">Your package has been delivered</p>
            <p className="text-xs text-green-600">Please confirm delivery to release escrow funds to the seller</p>
            <button
              onClick={onConfirmDelivery}
              disabled={!onConfirmDelivery}
              className="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3 rounded-xl transition-colors text-sm flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <CheckCircle2 size={18} /> Confirm Delivery
            </button>
          </div>
        )}
        {(isBuyer || !isBuyer) && (reachedStatus === 'in_transit' || reachedStatus === 'out_for_delivery' || reachedStatus === 'delivered') && (
          <button
            onClick={onUploadProof}
            disabled={!onUploadProof}
            className="w-full border-2 border-[#0071CE] text-[#0071CE] font-bold py-3 rounded-xl hover:bg-blue-50 transition-colors text-sm flex items-center justify-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Upload size={18} /> Upload Proof of Delivery
          </button>
        )}
      </div>
    </div>
  );
}
