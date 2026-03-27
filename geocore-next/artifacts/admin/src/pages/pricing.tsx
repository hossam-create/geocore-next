import { usePricingPlans, useSavePricingPlan } from "@/hooks/use-pricing";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CheckCircle2, Edit2, Plus } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

export default function PricingPage() {
  const { data: plans, isLoading } = usePricingPlans();
  const savePlan = useSavePricingPlan();
  const { toast } = useToast();

  if (isLoading) return <div>Loading plans...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold font-display tracking-tight text-foreground">Price Plans</h1>
        <Button className="shadow-sm">
          <Plus className="w-4 h-4 mr-2" /> Create Plan
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {plans?.map((plan: any) => (
          <Card key={plan.id} className="relative p-6 border-none shadow-sm flex flex-col hover:shadow-md transition-shadow">
            {plan.price === 99 && (
              <div className="absolute top-0 right-1/2 translate-x-1/2 -translate-y-1/2 bg-primary text-primary-foreground text-xs font-bold px-3 py-1 rounded-full uppercase tracking-widest shadow-sm">
                Most Popular
              </div>
            )}
            
            <div className="text-center mb-6 mt-2">
              <h3 className="text-xl font-bold font-display mb-1">{plan.name}</h3>
              <p className="text-muted-foreground text-sm mb-4">Group: {plan.group}</p>
              <div className="flex items-end justify-center gap-1">
                <span className="text-4xl font-black font-display text-foreground tracking-tight">AED {plan.price}</span>
                <span className="text-muted-foreground font-medium mb-1">/mo</span>
              </div>
            </div>

            <div className="space-y-4 flex-1 mb-8">
              <div className="flex items-center gap-3">
                <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
                <span className="text-sm font-medium text-foreground">{plan.max_listings} active listings</span>
              </div>
              <div className="flex items-center gap-3">
                <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
                <span className="text-sm font-medium text-foreground">{plan.max_images} images per listing</span>
              </div>
              <div className="flex items-center gap-3">
                <CheckCircle2 className={`w-5 h-5 shrink-0 ${plan.featured_allowed ? 'text-emerald-500' : 'text-muted-foreground/30'}`} />
                <span className={`text-sm font-medium ${plan.featured_allowed ? 'text-foreground' : 'text-muted-foreground line-through'}`}>Featured listings</span>
              </div>
              <div className="flex items-center gap-3">
                <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
                <span className="text-sm font-medium text-foreground">{plan.final_value_fee}% final value fee</span>
              </div>
            </div>

            <Button variant="outline" className="w-full border-border/50 bg-background hover:bg-muted font-semibold">
              <Edit2 className="w-4 h-4 mr-2" /> Edit Plan
            </Button>
          </Card>
        ))}
      </div>
    </div>
  );
}
