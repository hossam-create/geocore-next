-- Custom fields per category (dynamic attributes like year, mileage, area, etc.)

CREATE TABLE IF NOT EXISTS category_fields (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    label        VARCHAR(100) NOT NULL,
    label_en     VARCHAR(100),
    field_type   VARCHAR(20) NOT NULL CHECK (
        field_type IN ('text','number','select','boolean','range','date')
    ),
    options      JSONB DEFAULT '[]',
    is_required  BOOLEAN DEFAULT FALSE,
    placeholder  VARCHAR(200),
    unit         VARCHAR(20),
    sort_order   INTEGER DEFAULT 0,
    is_active    BOOLEAN DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cf_category ON category_fields(category_id) WHERE is_active = TRUE;

-- Add custom_fields column to listings
ALTER TABLE listings ADD COLUMN IF NOT EXISTS custom_fields JSONB DEFAULT '{}';

-- Seed: Vehicles (سيارات)
INSERT INTO category_fields (category_id, name, label, label_en, field_type, options, is_required, unit, sort_order)
SELECT c.id, f.name, f.label, f.label_en, f.field_type, f.options::jsonb, f.is_required, f.unit, f.sort_order
FROM categories c
CROSS JOIN (VALUES
    ('year',         'سنة الصنع',    'Year',         'number', '[]', true,  NULL,  1),
    ('mileage',      'الكيلومتراج',  'Mileage',      'number', '[]', false, 'km',  2),
    ('transmission', 'ناقل الحركة',  'Transmission',  'select',
        '[{"value":"manual","label":"مانيوال"},{"value":"automatic","label":"أوتوماتيك"}]',
        false, NULL, 3),
    ('fuel_type',    'نوع الوقود',   'Fuel Type',     'select',
        '[{"value":"petrol","label":"بنزين"},{"value":"diesel","label":"ديزل"},{"value":"electric","label":"كهربائي"},{"value":"hybrid","label":"هايبريد"}]',
        false, NULL, 4),
    ('engine_size',  'سعة المحرك',   'Engine Size',   'number', '[]', false, 'cc',  5),
    ('color',        'اللون',        'Color',         'text',   '[]', false, NULL,  6)
) AS f(name, label, label_en, field_type, options, is_required, unit, sort_order)
WHERE c.slug = 'vehicles';

-- Seed: Real Estate (عقارات)
INSERT INTO category_fields (category_id, name, label, label_en, field_type, options, is_required, unit, sort_order)
SELECT c.id, f.name, f.label, f.label_en, f.field_type, f.options::jsonb, f.is_required, f.unit, f.sort_order
FROM categories c
CROSS JOIN (VALUES
    ('area',       'المساحة',       'Area',       'number',  '[]', true,  'm²',  1),
    ('rooms',      'عدد الغرف',     'Rooms',      'number',  '[]', false, NULL,  2),
    ('bathrooms',  'عدد الحمامات',  'Bathrooms',  'number',  '[]', false, NULL,  3),
    ('floor',      'الدور',         'Floor',      'number',  '[]', false, NULL,  4),
    ('furnished',  'مفروشة',        'Furnished',  'boolean', '[]', false, NULL,  5)
) AS f(name, label, label_en, field_type, options, is_required, unit, sort_order)
WHERE c.slug = 'real-estate';

-- Seed: Electronics (إلكترونيات)
INSERT INTO category_fields (category_id, name, label, label_en, field_type, options, is_required, unit, sort_order)
SELECT c.id, f.name, f.label, f.label_en, f.field_type, f.options::jsonb, f.is_required, f.unit, f.sort_order
FROM categories c
CROSS JOIN (VALUES
    ('brand',     'الماركة',   'Brand',     'text',   '[]', false, NULL, 1),
    ('model',     'الموديل',   'Model',     'text',   '[]', false, NULL, 2),
    ('condition', 'الحالة',    'Condition', 'select',
        '[{"value":"new","label":"جديد"},{"value":"like_new","label":"شبه جديد"},{"value":"used","label":"مستعمل"}]',
        false, NULL, 3),
    ('warranty',  'الضمان',    'Warranty',  'boolean', '[]', false, NULL, 4)
) AS f(name, label, label_en, field_type, options, is_required, unit, sort_order)
WHERE c.slug = 'electronics';
