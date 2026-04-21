"use client";

import type { CategoryField } from "@/hooks/useCategoryFields";

interface Props {
  fields: CategoryField[];
  values: Record<string, string>;
  onChange: (values: Record<string, string>) => void;
}

export function DynamicFieldsForm({ fields, values, onChange }: Props) {
  const set = (name: string, value: string) => {
    onChange({ ...values, [name]: value });
  };

  if (!fields.length) return null;

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-gray-700">Category Details</h3>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {fields.map((field) => {
          const val = values[field.name] ?? "";
          let options: { value: string; label: string }[] = [];
          try {
            options = typeof field.options === "string" ? JSON.parse(field.options) : field.options;
          } catch { /* empty */ }

          return (
            <div key={field.id}>
              <label className="block text-xs font-medium text-gray-600 mb-1">
                {field.label_en || field.label}
                {field.is_required && <span className="text-red-500 ml-0.5">*</span>}
                {field.unit && (
                  <span className="text-gray-400 ml-1">({field.unit})</span>
                )}
              </label>

              {field.field_type === "text" && (
                <input
                  type="text"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={val}
                  onChange={(e) => set(field.name, e.target.value)}
                  placeholder={field.placeholder || field.label_en || field.label}
                />
              )}

              {field.field_type === "number" && (
                <input
                  type="number"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={val}
                  onChange={(e) => set(field.name, e.target.value)}
                  placeholder={field.placeholder || field.label_en || field.label}
                />
              )}

              {field.field_type === "select" && (
                <select
                  className="w-full border rounded-lg px-3 py-2 text-sm bg-white"
                  value={val}
                  onChange={(e) => set(field.name, e.target.value)}
                >
                  <option value="">Select...</option>
                  {options.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              )}

              {field.field_type === "boolean" && (
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={val === "true"}
                    onChange={(e) => set(field.name, e.target.checked ? "true" : "false")}
                    className="w-4 h-4 accent-purple-600"
                  />
                  <span className="text-sm text-gray-600">
                    {field.label_en || field.label}
                  </span>
                </label>
              )}

              {field.field_type === "date" && (
                <input
                  type="date"
                  className="w-full border rounded-lg px-3 py-2 text-sm"
                  value={val}
                  onChange={(e) => set(field.name, e.target.value)}
                />
              )}

              {field.field_type === "range" && (
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    className="w-full border rounded-lg px-3 py-2 text-sm"
                    value={val.split("-")[0] ?? ""}
                    onChange={(e) => {
                      const max = val.split("-")[1] ?? "";
                      set(field.name, `${e.target.value}-${max}`);
                    }}
                    placeholder="Min"
                  />
                  <span className="text-gray-400 text-xs">–</span>
                  <input
                    type="number"
                    className="w-full border rounded-lg px-3 py-2 text-sm"
                    value={val.split("-")[1] ?? ""}
                    onChange={(e) => {
                      const min = val.split("-")[0] ?? "";
                      set(field.name, `${min}-${e.target.value}`);
                    }}
                    placeholder="Max"
                  />
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
