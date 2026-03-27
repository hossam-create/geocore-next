export interface FieldDefinition {
  name: string;
  label: string;
  labelAr: string;
  fieldType: "text" | "number" | "select" | "boolean";
  unit?: string;
  filterType?: "text" | "number_range" | "select";
  filterOptions?: { label: string; value: string }[];
}

export interface CategoryFieldSchema {
  slug: string;
  fields: FieldDefinition[];
  keyFields: string[];
}

export const CATEGORY_FIELDS: Record<string, CategoryFieldSchema> = {
  vehicles: {
    slug: "vehicles",
    fields: [
      { name: "make", label: "Make", labelAr: "الماركة", fieldType: "text", filterType: "text" },
      { name: "model", label: "Model", labelAr: "الموديل", fieldType: "text", filterType: "text" },
      { name: "year", label: "Year", labelAr: "السنة", fieldType: "number", filterType: "number_range" },
      { name: "mileage", label: "Mileage", labelAr: "المسافة المقطوعة", fieldType: "number", unit: "km", filterType: "number_range" },
      { name: "color", label: "Color", labelAr: "اللون", fieldType: "text" },
      { name: "engine", label: "Engine", labelAr: "المحرك", fieldType: "text" },
    ],
    keyFields: ["year", "mileage"],
  },
  "real-estate": {
    slug: "real-estate",
    fields: [
      { name: "bedrooms", label: "Bedrooms", labelAr: "غرف النوم", fieldType: "number", filterType: "number_range" },
      { name: "bathrooms", label: "Bathrooms", labelAr: "دورات المياه", fieldType: "number", filterType: "number_range" },
      { name: "area", label: "Area", labelAr: "المساحة", fieldType: "number", unit: "sqm", filterType: "number_range" },
      {
        name: "furnished",
        label: "Furnished",
        labelAr: "مفروش",
        fieldType: "select",
        filterType: "select",
        filterOptions: [
          { label: "Furnished", value: "yes" },
          { label: "Unfurnished", value: "no" },
          { label: "Semi-Furnished", value: "semi" },
        ],
      },
      { name: "floor", label: "Floor", labelAr: "الطابق", fieldType: "number" },
    ],
    keyFields: ["bedrooms", "area"],
  },
  electronics: {
    slug: "electronics",
    fields: [
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "storage", label: "Storage", labelAr: "التخزين", fieldType: "text", filterType: "text" },
      { name: "ram", label: "RAM", labelAr: "الذاكرة العشوائية", fieldType: "text" },
      { name: "warranty", label: "Warranty", labelAr: "الضمان", fieldType: "text" },
    ],
    keyFields: ["brand", "storage"],
  },
  jobs: {
    slug: "jobs",
    fields: [
      {
        name: "job_type",
        label: "Job Type",
        labelAr: "نوع الوظيفة",
        fieldType: "text",
        filterType: "text",
      },
      { name: "salary_range", label: "Salary Range", labelAr: "نطاق الراتب", fieldType: "text" },
      { name: "experience_required", label: "Experience Required", labelAr: "الخبرة المطلوبة", fieldType: "text" },
      {
        name: "work_type",
        label: "Work Type",
        labelAr: "طريقة العمل",
        fieldType: "select",
        filterType: "select",
        filterOptions: [
          { label: "Remote", value: "remote" },
          { label: "On-site", value: "onsite" },
          { label: "Hybrid", value: "hybrid" },
        ],
      },
    ],
    keyFields: ["job_type", "work_type"],
  },
  furniture: {
    slug: "furniture",
    fields: [
      { name: "material", label: "Material", labelAr: "المادة", fieldType: "text" },
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "color", label: "Color", labelAr: "اللون", fieldType: "text" },
      { name: "dimensions", label: "Dimensions", labelAr: "الأبعاد", fieldType: "text" },
    ],
    keyFields: ["material", "brand"],
  },
  clothing: {
    slug: "clothing",
    fields: [
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "size", label: "Size", labelAr: "المقاس", fieldType: "text", filterType: "text" },
      { name: "color", label: "Color", labelAr: "اللون", fieldType: "text" },
      { name: "material", label: "Material", labelAr: "المادة", fieldType: "text" },
    ],
    keyFields: ["brand", "size"],
  },
  jewelry: {
    slug: "jewelry",
    fields: [
      { name: "material", label: "Material", labelAr: "المادة", fieldType: "text", filterType: "text" },
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "weight", label: "Weight", labelAr: "الوزن", fieldType: "text", unit: "g" },
      { name: "gemstone", label: "Gemstone", labelAr: "الحجر الكريم", fieldType: "text" },
    ],
    keyFields: ["material", "brand"],
  },
  gaming: {
    slug: "gaming",
    fields: [
      { name: "platform", label: "Platform", labelAr: "المنصة", fieldType: "text", filterType: "text" },
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "storage", label: "Storage", labelAr: "التخزين", fieldType: "text" },
      { name: "warranty", label: "Warranty", labelAr: "الضمان", fieldType: "text" },
    ],
    keyFields: ["platform", "brand"],
  },
  sports: {
    slug: "sports",
    fields: [
      { name: "brand", label: "Brand", labelAr: "العلامة التجارية", fieldType: "text", filterType: "text" },
      { name: "sport_type", label: "Sport Type", labelAr: "نوع الرياضة", fieldType: "text", filterType: "text" },
      { name: "size", label: "Size", labelAr: "المقاس", fieldType: "text" },
      { name: "material", label: "Material", labelAr: "المادة", fieldType: "text" },
    ],
    keyFields: ["brand", "sport_type"],
  },
  animals: {
    slug: "animals",
    fields: [
      { name: "species", label: "Species", labelAr: "النوع", fieldType: "text", filterType: "text" },
      { name: "breed", label: "Breed", labelAr: "السلالة", fieldType: "text", filterType: "text" },
      { name: "age", label: "Age", labelAr: "العمر", fieldType: "text" },
      { name: "gender", label: "Gender", labelAr: "الجنس", fieldType: "text" },
    ],
    keyFields: ["species", "breed"],
  },
};

export function getCategorySchema(categorySlug?: string): CategoryFieldSchema | null {
  if (!categorySlug) return null;
  return CATEGORY_FIELDS[categorySlug] ?? null;
}

export function formatFieldValue(field: FieldDefinition, rawValue: unknown): string | null {
  if (rawValue === null || rawValue === undefined || rawValue === "") return null;
  const val = String(rawValue);
  if (field.unit) return `${val} ${field.unit}`;
  return val;
}

export function getKeyAttributePills(
  categorySlug: string | undefined,
  attributes: Record<string, unknown> | null | undefined
): string[] {
  if (!categorySlug || !attributes) return [];
  const schema = getCategorySchema(categorySlug);
  if (!schema) return [];

  return schema.keyFields
    .map((fname) => {
      const field = schema.fields.find((f) => f.name === fname);
      if (!field) return null;
      const raw = attributes[fname];
      return formatFieldValue(field, raw);
    })
    .filter((v): v is string => v !== null);
}
