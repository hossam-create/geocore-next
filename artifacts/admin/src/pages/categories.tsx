import { useState } from "react";
import { useCategories, useCategoryActions } from "@/hooks/use-categories";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2, Edit2, Settings2 } from "lucide-react";

export default function CategoriesPage() {
  const { data: categories, isLoading } = useCategories();
  const [selectedId, setSelectedId] = useState<string | null>("1");
  const actions = useCategoryActions();

  const selectedCat = categories?.find((c: any) => c.id === selectedId);

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="space-y-6 h-[calc(100vh-8rem)] flex flex-col">
      <div className="flex items-center justify-between shrink-0">
        <h1 className="text-3xl font-bold font-display tracking-tight text-foreground">Categories & Fields</h1>
        <Button><Plus className="w-4 h-4 mr-2" /> New Category</Button>
      </div>

      <div className="flex gap-6 flex-1 min-h-0">
        {/* Left Sidebar - Tree */}
        <Card className="w-1/3 p-4 border-none shadow-sm flex flex-col h-full">
          <Input placeholder="Search categories..." className="mb-4 bg-muted/50" />
          <div className="space-y-1 overflow-y-auto pr-2">
            {categories?.map((cat: any) => (
              <button
                key={cat.id}
                onClick={() => setSelectedId(cat.id)}
                className={`w-full flex items-center gap-3 px-3 py-3 rounded-xl transition-colors ${
                  selectedId === cat.id 
                    ? "bg-primary/10 text-primary font-semibold" 
                    : "hover:bg-muted text-foreground font-medium"
                }`}
              >
                <span className="text-xl bg-background rounded-md w-8 h-8 flex items-center justify-center shadow-sm">{cat.icon}</span>
                <span className="flex-1 text-left">{cat.name_en}</span>
                {!cat.active && <Badge variant="secondary" className="text-[10px]">Hidden</Badge>}
              </button>
            ))}
          </div>
        </Card>

        {/* Right Panel - Details & Fields */}
        <Card className="w-2/3 p-6 border-none shadow-sm overflow-y-auto h-full relative">
          {selectedCat ? (
            <div className="space-y-8 max-w-2xl animate-in fade-in slide-in-from-right-4 duration-300">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-4">
                  <div className="text-4xl bg-muted rounded-2xl w-16 h-16 flex items-center justify-center">{selectedCat.icon}</div>
                  <div>
                    <h2 className="text-2xl font-bold font-display text-foreground">{selectedCat.name_en}</h2>
                    <p className="text-muted-foreground">{selectedCat.name_ar}</p>
                  </div>
                </div>
                <Button variant="outline" size="sm"><Edit2 className="w-4 h-4 mr-2" /> Edit Info</Button>
              </div>

              <div>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-bold font-display text-foreground flex items-center gap-2">
                    <Settings2 className="w-5 h-5 text-primary" /> Custom Fields
                  </h3>
                  <Button size="sm" variant="secondary"><Plus className="w-4 h-4 mr-2" /> Add Field</Button>
                </div>
                
                {selectedCat.custom_fields?.length > 0 ? (
                  <div className="space-y-3">
                    {selectedCat.custom_fields.map((field: any) => (
                      <div key={field.id} className="border border-border/50 bg-muted/20 p-4 rounded-xl flex items-center gap-4 group">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-semibold text-foreground">{field.label_en}</span>
                            <Badge variant="outline" className="bg-background text-xs uppercase">{field.type}</Badge>
                            {field.required && <Badge className="bg-destructive/10 text-destructive text-[10px] hover:bg-destructive/10">Required</Badge>}
                          </div>
                          <p className="text-xs font-mono text-muted-foreground bg-muted w-fit px-1.5 py-0.5 rounded">{field.key}</p>
                          {field.options && <p className="text-xs text-muted-foreground mt-2 truncate">Options: {field.options}</p>}
                        </div>
                        <Button size="icon" variant="ghost" className="text-destructive opacity-0 group-hover:opacity-100 transition-opacity">
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center p-8 bg-muted/20 rounded-xl border border-dashed border-border">
                    <p className="text-muted-foreground">No custom fields defined for this category.</p>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              Select a category to view details
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
