import { useState, useEffect } from "react";
import { useLocation } from "wouter";
import { useAuthStore } from "@/store/auth";
import api from "@/lib/api";
import { ChevronRight, ChevronLeft, Check, Eye } from "lucide-react";
import { ImageUploader } from "@/components/ui/ImageUploader";

const LISTING_TYPES = [
  { id: "buy_now", label: "Buy Now", desc: "Fixed price — buyer pays immediately" },
  { id: "standard_auction", label: "Standard Auction", desc: "Highest bidder wins at deadline" },
  { id: "dutch_auction", label: "Dutch Auction", desc: "Price drops over time until sold" },
  { id: "reverse_auction", label: "Reverse Auction", desc: "Sellers compete to offer lowest price" },
];

const CATEGORIES = [
  { id: "vehicles", label: "🚗 Vehicles" },
  { id: "real-estate", label: "🏠 Real Estate" },
  { id: "electronics", label: "📱 Electronics" },
  { id: "jobs", label: "💼 Jobs" },
  { id: "furniture", label: "🛋️ Furniture" },
  { id: "clothing", label: "👕 Clothing" },
  { id: "jewelry", label: "💍 Jewelry" },
  { id: "gaming", label: "🎮 Gaming" },
  { id: "sports", label: "⚽ Sports" },
  { id: "animals", label: "🐾 Animals" },
];

const CATEGORY_FIELDS: Record<string, { key: string; label: string; type?: string; placeholder?: string; options?: string[] }[]> = {
  vehicles: [
    { key: "make", label: "Make", placeholder: "e.g. Toyota" },
    { key: "model", label: "Model", placeholder: "e.g. Camry" },
    { key: "year", label: "Year", type: "number", placeholder: "e.g. 2022" },
    { key: "mileage", label: "Mileage (km)", type: "number", placeholder: "e.g. 45000" },
  ],
  "real-estate": [
    { key: "bedrooms", label: "Bedrooms", type: "number", placeholder: "e.g. 3" },
    { key: "bathrooms", label: "Bathrooms", type: "number", placeholder: "e.g. 2" },
    { key: "area_sqm", label: "Area (m²)", type: "number", placeholder: "e.g. 120" },
    { key: "furnished", label: "Furnished", options: ["Yes", "No", "Partially"] },
  ],
  electronics: [
    { key: "brand", label: "Brand", placeholder: "e.g. Apple" },
    { key: "storage", label: "Storage", placeholder: "e.g. 256GB" },
    { key: "ram", label: "RAM", placeholder: "e.g. 16GB" },
  ],
  jobs: [
    { key: "job_type", label: "Job Type", options: ["Full-Time", "Part-Time", "Contract", "Freelance", "Internship"] },
    { key: "salary", label: "Salary / Pay Rate", placeholder: "e.g. $50,000/year" },
    { key: "experience", label: "Experience Required", placeholder: "e.g. 2+ years" },
  ],
  furniture: [
    { key: "material", label: "Material", placeholder: "e.g. Wood, Metal" },
    { key: "dimensions", label: "Dimensions", placeholder: "e.g. 200x90x75 cm" },
  ],
  clothing: [
    { key: "size", label: "Size", placeholder: "e.g. M, L, XL" },
    { key: "brand", label: "Brand", placeholder: "e.g. Nike" },
    { key: "color", label: "Color", placeholder: "e.g. Blue" },
  ],
  jewelry: [
    { key: "material", label: "Material", placeholder: "e.g. Gold, Silver" },
    { key: "gemstone", label: "Gemstone", placeholder: "e.g. Diamond (optional)" },
  ],
  gaming: [
    { key: "platform", label: "Platform", options: ["PC", "PlayStation", "Xbox", "Nintendo Switch", "Other"] },
    { key: "genre", label: "Genre", placeholder: "e.g. Action, RPG" },
  ],
  sports: [
    { key: "sport", label: "Sport", placeholder: "e.g. Football, Tennis" },
    { key: "brand", label: "Brand", placeholder: "e.g. Adidas" },
  ],
  animals: [
    { key: "species", label: "Species", placeholder: "e.g. Dog, Cat" },
    { key: "breed", label: "Breed", placeholder: "e.g. Labrador" },
    { key: "age", label: "Age", placeholder: "e.g. 2 years" },
  ],
};

const CONDITIONS = ["New", "Like New", "Good", "Fair", "For Parts"];
const DURATIONS = [
  { value: 1, label: "1 day" },
  { value: 3, label: "3 days" },
  { value: 7, label: "7 days" },
  { value: 14, label: "14 days" },
  { value: 30, label: "30 days" },
];

interface FormState {
  listingType: string;
  category: string;
  title: string;
  description: string;
  price: string;
  startBid: string;
  reservePrice: string;
  buyNowPrice: string;
  maxBudget: string;
  deadline: string;
  condition: string;
  city: string;
  country: string;
  uploadedImages: { key: string; url: string; file_name: string }[];
  duration: number;
  featured: boolean;
  quantity: string;
  attributes: Record<string, string>;
}

const INITIAL_FORM: FormState = {
  listingType: "",
  category: "",
  title: "",
  description: "",
  price: "",
  startBid: "",
  reservePrice: "",
  buyNowPrice: "",
  maxBudget: "",
  deadline: "",
  condition: "Good",
  city: "",
  country: "",
  uploadedImages: [],
  duration: 7,
  featured: false,
  quantity: "1",
  attributes: {},
};

function FieldError({ msg }: { msg?: string }) {
  if (!msg) return null;
  return <p className="text-red-500 text-xs mt-1">{msg}</p>;
}

function InputField({
  label, value, onChange, placeholder, type = "text", required, error, min,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  required?: boolean;
  error?: string;
  min?: string;
}) {
  return (
    <div>
      <label className="text-sm font-medium text-gray-700 block mb-1.5">
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        min={min}
        className={`w-full border rounded-lg px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent transition ${
          error ? "border-red-400" : "border-gray-200"
        }`}
      />
      <FieldError msg={error} />
    </div>
  );
}

function SelectField({
  label, value, onChange, options, required, error,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: string[];
  required?: boolean;
  error?: string;
}) {
  return (
    <div>
      <label className="text-sm font-medium text-gray-700 block mb-1.5">
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={`w-full border rounded-lg px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent bg-white transition ${
          error ? "border-red-400" : "border-gray-200"
        }`}
      >
        <option value="">Select…</option>
        {options.map((o) => (
          <option key={o} value={o}>{o}</option>
        ))}
      </select>
      <FieldError msg={error} />
    </div>
  );
}

export default function SellPage() {
  const { isAuthenticated } = useAuthStore();
  const [, navigate] = useLocation();
  const [step, setStep] = useState(1);
  const [form, setForm] = useState<FormState>(INITIAL_FORM);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      navigate("/login?next=/sell");
    }
  }, [isAuthenticated]);

  const setField = (key: keyof FormState) => (value: string | boolean | number) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[key as string];
      return next;
    });
  };

  const setAttr = (key: string, value: string) => {
    setForm((prev) => ({ ...prev, attributes: { ...prev.attributes, [key]: value } }));
  };

  const validateStep1 = (): boolean => {
    const errs: Record<string, string> = {};
    if (!form.listingType) errs.listingType = "Please choose a listing type.";
    if (!form.category) errs.category = "Please choose a category.";
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const validateStep2 = (): boolean => {
    const errs: Record<string, string> = {};
    if (!form.title.trim()) errs.title = "Title is required.";
    if (!form.description.trim()) errs.description = "Description is required.";
    if (!form.city.trim()) errs.city = "City is required.";
    if (!form.country.trim()) errs.country = "Country is required.";

    if (form.listingType === "buy_now") {
      if (!form.price || Number(form.price) <= 0) errs.price = "Price must be greater than 0.";
    } else if (form.listingType === "reverse_auction") {
      if (!form.maxBudget || Number(form.maxBudget) <= 0) errs.maxBudget = "Max budget must be greater than 0.";
    } else {
      if (!form.startBid || Number(form.startBid) <= 0) errs.startBid = "Starting bid must be greater than 0.";
    }
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleNext = () => {
    if (step === 1 && validateStep1()) setStep(2);
    else if (step === 2 && validateStep2()) setStep(3);
  };

  const handleBack = () => setStep((s) => Math.max(1, s - 1));

  const handleSubmit = async () => {
    setSubmitError("");
    setIsSubmitting(true);
    try {
      const images = form.uploadedImages.map((img) => img.url);
      const isAuction = form.listingType !== "buy_now";

      if (isAuction) {
        const payload: Record<string, unknown> = {
          title: form.title,
          description: form.description,
          category: form.category,
          auction_type: form.listingType,
          condition: form.condition,
          location: `${form.city}, ${form.country}`,
          images,
          start_price: Number(form.startBid) || 0,
          reserve_price: form.reservePrice ? Number(form.reservePrice) : undefined,
          buy_now_price: form.buyNowPrice ? Number(form.buyNowPrice) : undefined,
          duration_hours: form.duration * 24,
          featured: form.featured,
          attributes: form.attributes,
        };
        if (form.listingType === "dutch_auction") {
          payload.quantity = Number(form.quantity) || 1;
        }
        if (form.listingType === "reverse_auction") {
          payload.max_budget = Number(form.maxBudget);
          payload.deadline = form.deadline || undefined;
        }
        const { data } = await api.post("/auctions", payload);
        const newId = data?.data?.id || data?.data?.auction?.id;
        navigate(newId ? `/auctions/${newId}` : "/auctions");
      } else {
        const payload: Record<string, unknown> = {
          title: form.title,
          description: form.description,
          category: form.category,
          price: Number(form.price),
          condition: form.condition,
          location: `${form.city}, ${form.country}`,
          images,
          featured: form.featured,
          attributes: form.attributes,
        };
        const { data } = await api.post("/listings", payload);
        const newId = data?.data?.id || data?.data?.listing?.id;
        navigate(newId ? `/listings/${newId}` : "/listings");
      }
    } catch (err: any) {
      const msg =
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        "Submission failed. Please try again.";
      setSubmitError(msg);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!isAuthenticated) return null;

  const isAuction = form.listingType !== "buy_now" && form.listingType !== "";
  const isDutch = form.listingType === "dutch_auction";
  const isReverse = form.listingType === "reverse_auction";
  const catFields = CATEGORY_FIELDS[form.category] || [];

  const steps = ["Type & Category", "Details", "Options"];

  return (
    <div className="min-h-[80vh] bg-gray-50 py-8 px-4">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-2xl font-extrabold text-gray-900 mb-6">Post a Listing</h1>

        <div className="flex items-center gap-2 mb-8">
          {steps.map((label, idx) => {
            const num = idx + 1;
            const active = step === num;
            const done = step > num;
            return (
              <div key={num} className="flex items-center gap-2 flex-1">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${
                  done ? "bg-green-500 text-white" : active ? "bg-[#0071CE] text-white" : "bg-gray-200 text-gray-500"
                }`}>
                  {done ? <Check size={13} /> : num}
                </div>
                <span className={`text-xs font-medium whitespace-nowrap ${active ? "text-[#0071CE]" : "text-gray-400"}`}>{label}</span>
                {idx < steps.length - 1 && <div className="flex-1 h-px bg-gray-200 mx-2" />}
              </div>
            );
          })}
        </div>

        <div className="bg-white rounded-2xl shadow-sm p-6 space-y-5">
          {step === 1 && (
            <>
              <div>
                <p className="text-sm font-semibold text-gray-700 mb-3">
                  Listing Type <span className="text-red-500">*</span>
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {LISTING_TYPES.map((t) => (
                    <button
                      key={t.id}
                      onClick={() => setField("listingType")(t.id)}
                      className={`text-left border rounded-xl px-4 py-3 transition ${
                        form.listingType === t.id
                          ? "border-[#0071CE] bg-blue-50 ring-2 ring-[#0071CE]"
                          : "border-gray-200 hover:border-gray-300"
                      }`}
                    >
                      <div className="font-semibold text-sm text-gray-800">{t.label}</div>
                      <div className="text-xs text-gray-500 mt-0.5">{t.desc}</div>
                    </button>
                  ))}
                </div>
                <FieldError msg={errors.listingType} />
              </div>

              <div>
                <p className="text-sm font-semibold text-gray-700 mb-3">
                  Category <span className="text-red-500">*</span>
                </p>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {CATEGORIES.map((c) => (
                    <button
                      key={c.id}
                      onClick={() => setField("category")(c.id)}
                      className={`text-left border rounded-xl px-3 py-2.5 text-sm transition ${
                        form.category === c.id
                          ? "border-[#0071CE] bg-blue-50 ring-2 ring-[#0071CE] font-semibold text-[#0071CE]"
                          : "border-gray-200 hover:border-gray-300 text-gray-700"
                      }`}
                    >
                      {c.label}
                    </button>
                  ))}
                </div>
                <FieldError msg={errors.category} />
              </div>
            </>
          )}

          {step === 2 && (
            <>
              <InputField
                label="Title"
                value={form.title}
                onChange={setField("title") as (v: string) => void}
                placeholder="What are you selling?"
                required
                error={errors.title}
              />
              <div>
                <label className="text-sm font-medium text-gray-700 block mb-1.5">
                  Description <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={form.description}
                  onChange={(e) => {
                    setField("description")(e.target.value);
                  }}
                  placeholder="Describe your item in detail..."
                  rows={4}
                  className={`w-full border rounded-lg px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent transition resize-none ${
                    errors.description ? "border-red-400" : "border-gray-200"
                  }`}
                />
                <FieldError msg={errors.description} />
              </div>

              {form.listingType === "buy_now" && (
                <InputField
                  label="Price (USD)"
                  value={form.price}
                  onChange={setField("price") as (v: string) => void}
                  type="number"
                  placeholder="e.g. 299.99"
                  required
                  error={errors.price}
                  min="0"
                />
              )}

              {(form.listingType === "standard_auction" || form.listingType === "dutch_auction") && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <InputField
                    label="Starting Bid (USD)"
                    value={form.startBid}
                    onChange={setField("startBid") as (v: string) => void}
                    type="number"
                    placeholder="e.g. 50"
                    required
                    error={errors.startBid}
                    min="0"
                  />
                  <InputField
                    label="Reserve Price (USD, optional)"
                    value={form.reservePrice}
                    onChange={setField("reservePrice") as (v: string) => void}
                    type="number"
                    placeholder="e.g. 200"
                    min="0"
                  />
                  <InputField
                    label="Buy Now Price (USD, optional)"
                    value={form.buyNowPrice}
                    onChange={setField("buyNowPrice") as (v: string) => void}
                    type="number"
                    placeholder="e.g. 400"
                    min="0"
                  />
                </div>
              )}

              {form.listingType === "reverse_auction" && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <InputField
                    label="Max Budget (USD)"
                    value={form.maxBudget}
                    onChange={setField("maxBudget") as (v: string) => void}
                    type="number"
                    placeholder="e.g. 1000"
                    required
                    error={errors.maxBudget}
                    min="0"
                  />
                  <InputField
                    label="Deadline (optional)"
                    value={form.deadline}
                    onChange={setField("deadline") as (v: string) => void}
                    type="datetime-local"
                  />
                </div>
              )}

              <SelectField
                label="Condition"
                value={form.condition}
                onChange={setField("condition") as (v: string) => void}
                options={CONDITIONS}
                required
              />

              <div className="grid grid-cols-2 gap-4">
                <InputField
                  label="City"
                  value={form.city}
                  onChange={setField("city") as (v: string) => void}
                  placeholder="e.g. New York"
                  required
                  error={errors.city}
                />
                <InputField
                  label="Country"
                  value={form.country}
                  onChange={setField("country") as (v: string) => void}
                  placeholder="e.g. USA"
                  required
                  error={errors.country}
                />
              </div>

              <div>
                <label className="text-sm font-medium text-gray-700 block mb-2">
                  Photos <span className="text-gray-400 font-normal">(up to 8)</span>
                </label>
                <ImageUploader
                  images={form.uploadedImages}
                  onChange={(imgs) => setForm((f) => ({ ...f, uploadedImages: imgs }))}
                  maxImages={8}
                  folder="listings"
                />
              </div>

              {catFields.length > 0 && (
                <div className="border-t pt-4">
                  <p className="text-sm font-semibold text-gray-700 mb-3">
                    {CATEGORIES.find((c) => c.id === form.category)?.label} Details
                  </p>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {catFields.map((f) =>
                      f.options ? (
                        <SelectField
                          key={f.key}
                          label={f.label}
                          value={form.attributes[f.key] || ""}
                          onChange={(v) => setAttr(f.key, v)}
                          options={f.options}
                        />
                      ) : (
                        <InputField
                          key={f.key}
                          label={f.label}
                          value={form.attributes[f.key] || ""}
                          onChange={(v) => setAttr(f.key, v)}
                          placeholder={f.placeholder}
                          type={f.type || "text"}
                        />
                      )
                    )}
                  </div>
                </div>
              )}
            </>
          )}

          {step === 3 && (
            <>
              {!isReverse && (
                <div>
                  <label className="text-sm font-medium text-gray-700 block mb-2">
                    Listing Duration
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {DURATIONS.map((d) => (
                      <button
                        key={d.value}
                        onClick={() => setField("duration")(d.value)}
                        className={`px-4 py-2 text-sm rounded-lg border transition ${
                          form.duration === d.value
                            ? "bg-[#0071CE] text-white border-[#0071CE]"
                            : "border-gray-200 text-gray-700 hover:border-gray-300"
                        }`}
                      >
                        {d.label}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {isDutch && (
                <InputField
                  label="Quantity"
                  value={form.quantity}
                  onChange={setField("quantity") as (v: string) => void}
                  type="number"
                  placeholder="e.g. 5"
                  min="1"
                />
              )}

              <div className="flex items-center gap-3 py-2">
                <input
                  id="featured"
                  type="checkbox"
                  checked={form.featured}
                  onChange={(e) => setField("featured")(e.target.checked)}
                  className="w-4 h-4 rounded border-gray-300 text-[#0071CE] cursor-pointer"
                />
                <label htmlFor="featured" className="text-sm font-medium text-gray-700 cursor-pointer">
                  Feature this listing (highlighted in search results)
                </label>
              </div>

              <div className="border rounded-xl p-4 bg-gray-50">
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">Preview</p>
                <h2 className="font-bold text-gray-800 text-lg">{form.title || "Your listing title"}</h2>
                <p className="text-gray-500 text-sm mt-1 line-clamp-3">{form.description || "Your description…"}</p>
                <div className="flex items-center gap-4 mt-3 flex-wrap">
                  {form.listingType === "buy_now" && form.price && (
                    <span className="text-green-600 font-bold">${Number(form.price).toLocaleString()}</span>
                  )}
                  {(form.listingType === "standard_auction" || form.listingType === "dutch_auction") && form.startBid && (
                    <span className="text-[#0071CE] font-bold">Starting at ${Number(form.startBid).toLocaleString()}</span>
                  )}
                  {form.listingType === "reverse_auction" && form.maxBudget && (
                    <span className="text-purple-600 font-bold">Budget: ${Number(form.maxBudget).toLocaleString()}</span>
                  )}
                  {form.condition && <span className="text-xs bg-gray-200 text-gray-600 px-2 py-0.5 rounded-full">{form.condition}</span>}
                  {(form.city || form.country) && (
                    <span className="text-xs text-gray-500">📍 {[form.city, form.country].filter(Boolean).join(", ")}</span>
                  )}
                </div>
                {form.uploadedImages.length > 0 && (
                  <div className="mt-3 grid grid-cols-3 gap-1.5">
                    {form.uploadedImages.slice(0, 3).map((img) => (
                      <img key={img.key} src={img.url} alt={img.file_name}
                        className="w-full h-24 object-cover rounded-lg" />
                    ))}
                  </div>
                )}
              </div>

              {submitError && (
                <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-3 text-sm">
                  {submitError}
                </div>
              )}
            </>
          )}
        </div>

        <div className="flex items-center justify-between mt-6">
          <button
            onClick={handleBack}
            disabled={step === 1}
            className="flex items-center gap-1 px-5 py-2.5 text-sm font-medium text-gray-600 bg-white border border-gray-200 rounded-xl hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
          >
            <ChevronLeft size={16} /> Back
          </button>

          {step < 3 ? (
            <button
              onClick={handleNext}
              className="flex items-center gap-1 px-6 py-2.5 text-sm font-bold bg-[#0071CE] text-white rounded-xl hover:bg-[#005BA1] transition"
            >
              Next <ChevronRight size={16} />
            </button>
          ) : (
            <button
              onClick={handleSubmit}
              disabled={isSubmitting}
              className="flex items-center gap-2 px-6 py-2.5 text-sm font-bold bg-[#FFC220] text-gray-900 rounded-xl hover:bg-yellow-400 disabled:opacity-60 disabled:cursor-not-allowed transition"
            >
              {isSubmitting ? "Posting…" : "Post Listing"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
