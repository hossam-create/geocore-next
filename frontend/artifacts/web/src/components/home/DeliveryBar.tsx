import { Truck, RotateCcw, ShieldCheck, BadgePercent } from "lucide-react";

const BENEFITS = [
  { icon: Truck, text: "Free delivery on orders over AED 200" },
  { icon: BadgePercent, text: "Daily deals & flash sales" },
  { icon: RotateCcw, text: "Easy 30-day returns" },
  { icon: ShieldCheck, text: "Buyer protection guaranteed" },
];

export function DeliveryBar() {
  return (
    <div className="bg-white border-b border-gray-100">
      <div className="max-w-7xl mx-auto px-4">
        <div className="grid grid-cols-2 md:grid-cols-4 divide-x divide-gray-100">
          {BENEFITS.map(({ icon: Icon, text }) => (
            <div key={text} className="flex items-center gap-2.5 py-2.5 px-4 justify-center">
              <Icon size={16} className="text-[#0071CE] shrink-0" />
              <span className="text-xs text-gray-600 font-medium">{text}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
