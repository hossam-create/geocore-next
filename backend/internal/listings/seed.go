package listings

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// seedCat is a helper to define a category with optional parent slug.
type seedCat struct {
	Slug      string
	NameEn    string
	NameAr    string
	Parent    string // parent slug (empty for L1)
	Color     string
	Icon      string
	IsLeaf    bool
	SortOrder int
}

// seedField defines a custom field for a category.
type seedField struct {
	CategorySlug string
	Name         string
	LabelEn      string
	LabelAr      string
	FieldType    string // text|number|select|boolean|date|range
	Options      []string
	IsRequired   bool
	SortOrder    int
}

func SeedCategories(db *gorm.DB) {
	var count int64
	db.Model(&Category{}).Count(&count)
	if count > 0 {
		return
	}

	// ════════════════════════════════════════════════════════════════════════════
	// LEVEL 1 — 12 Main Categories (eBay-style with colors)
	// ════════════════════════════════════════════════════════════════════════════
	cats := []seedCat{
		// 1. Electronics
		{Slug: "electronics", NameEn: "Electronics", NameAr: "إلكترونيات", Color: "#E53238", Icon: "📱", SortOrder: 1},
		{Slug: "phones-accessories", NameEn: "Cell Phones & Accessories", NameAr: "هواتف وإكسسوارات", Parent: "electronics", SortOrder: 1},
		{Slug: "computers-tablets", NameEn: "Computers, Tablets & More", NameAr: "كمبيوتر وتابلت", Parent: "electronics", SortOrder: 2},
		{Slug: "tv-audio-video", NameEn: "TV, Audio & Video", NameAr: "تليفزيون وصوت وصورة", Parent: "electronics", SortOrder: 3},
		{Slug: "cameras-photo", NameEn: "Cameras & Photo", NameAr: "كاميرات وتصوير", Parent: "electronics", SortOrder: 4},
		{Slug: "video-games-consoles", NameEn: "Video Games & Consoles", NameAr: "ألعاب فيديو", Parent: "electronics", SortOrder: 5},
		{Slug: "smart-home", NameEn: "Smart Home", NameAr: "المنزل الذكي", Parent: "electronics", SortOrder: 6},
		{Slug: "wearables", NameEn: "Wearables", NameAr: "الأجهزة القابلة للارتداء", Parent: "electronics", SortOrder: 7},
		// L3 under phones
		{Slug: "smartphones", NameEn: "Smartphones", NameAr: "هواتف ذكية", Parent: "phones-accessories", IsLeaf: true, SortOrder: 1},
		{Slug: "phone-cases", NameEn: "Cases & Covers", NameAr: "أغطية الهاتف", Parent: "phones-accessories", IsLeaf: true, SortOrder: 2},
		{Slug: "chargers", NameEn: "Chargers & Cables", NameAr: "شواحن وكابلات", Parent: "phones-accessories", IsLeaf: true, SortOrder: 3},
		{Slug: "headphones", NameEn: "Headphones", NameAr: "سماعات", Parent: "phones-accessories", IsLeaf: true, SortOrder: 4},

		// 2. Vehicles & Motors
		{Slug: "motors", NameEn: "Motors", NameAr: "سيارات ومركبات", Color: "#86B817", Icon: "🚗", SortOrder: 2},
		{Slug: "cars-trucks", NameEn: "Cars & Trucks", NameAr: "سيارات وشاحنات", Parent: "motors", SortOrder: 1},
		{Slug: "motorcycles", NameEn: "Motorcycles & ATVs", NameAr: "موتوسيكلات", Parent: "motors", SortOrder: 2},
		{Slug: "boats", NameEn: "Boats", NameAr: "قوارب", Parent: "motors", SortOrder: 3},
		{Slug: "auto-parts", NameEn: "Auto Parts & Vehicles", NameAr: "قطع غيار", Parent: "motors", SortOrder: 4},
		{Slug: "car-electronics", NameEn: "Car Electronics", NameAr: "إلكترونيات السيارة", Parent: "motors", SortOrder: 5},
		{Slug: "car-care", NameEn: "Car Care & Detailing", NameAr: "العناية بالسيارة", Parent: "motors", SortOrder: 6},
		// L3 under cars-trucks
		{Slug: "sedan", NameEn: "Sedans", NameAr: "سيدان", Parent: "cars-trucks", IsLeaf: true, SortOrder: 1},
		{Slug: "suv", NameEn: "SUVs", NameAr: "دفع رباعي", Parent: "cars-trucks", IsLeaf: true, SortOrder: 2},
		{Slug: "pickup", NameEn: "Pickups", NameAr: "بيك أب", Parent: "cars-trucks", IsLeaf: true, SortOrder: 3},
		{Slug: "van", NameEn: "Vans", NameAr: "فانات", Parent: "cars-trucks", IsLeaf: true, SortOrder: 4},

		// 3. Fashion & Apparel
		{Slug: "fashion", NameEn: "Fashion", NameAr: "أزياء", Color: "#F5AF02", Icon: "👗", SortOrder: 3},
		{Slug: "mens-clothing", NameEn: "Men's Clothing", NameAr: "ملابس رجالية", Parent: "fashion", SortOrder: 1},
		{Slug: "womens-clothing", NameEn: "Women's Clothing", NameAr: "ملابس نسائية", Parent: "fashion", SortOrder: 2},
		{Slug: "kids-clothing", NameEn: "Kids' Clothing", NameAr: "ملابس أطفال", Parent: "fashion", SortOrder: 3},
		{Slug: "shoes", NameEn: "Shoes", NameAr: "أحذية", Parent: "fashion", SortOrder: 4},
		{Slug: "bags-handbags", NameEn: "Bags & Handbags", NameAr: "حقائب", Parent: "fashion", SortOrder: 5},
		{Slug: "jewelry", NameEn: "Jewelry & Watches", NameAr: "مجوهرات وساعات", Parent: "fashion", SortOrder: 6},
		{Slug: "sportswear", NameEn: "Sportswear", NameAr: "ملابس رياضية", Parent: "fashion", SortOrder: 7},

		// 4. Home & Garden
		{Slug: "home-garden", NameEn: "Home & Garden", NameAr: "منزل وحديقة", Color: "#86B817", Icon: "🏠", SortOrder: 4},
		{Slug: "furniture", NameEn: "Furniture", NameAr: "أثاث", Parent: "home-garden", SortOrder: 1},
		{Slug: "kitchen", NameEn: "Kitchen & Dining", NameAr: "مطبخ وطعام", Parent: "home-garden", SortOrder: 2},
		{Slug: "bedding", NameEn: "Bedding & Linens", NameAr: "مفارش وبياضات", Parent: "home-garden", SortOrder: 3},
		{Slug: "appliances", NameEn: "Major Appliances", NameAr: "أجهزة كهربائية", Parent: "home-garden", SortOrder: 4},
		{Slug: "garden", NameEn: "Garden & Patio", NameAr: "حديقة وتراس", Parent: "home-garden", SortOrder: 5},
		{Slug: "home-decor", NameEn: "Home Décor", NameAr: "ديكور المنزل", Parent: "home-garden", SortOrder: 6},
		{Slug: "tools", NameEn: "Tools & Workshop", NameAr: "أدوات وورشة عمل", Parent: "home-garden", SortOrder: 7},

		// 5. Real Estate
		{Slug: "real-estate", NameEn: "Real Estate", NameAr: "عقارات", Color: "#0064D2", Icon: "🏢", SortOrder: 5},
		{Slug: "apartments-sale", NameEn: "Apartments for Sale", NameAr: "شقق للبيع", Parent: "real-estate", SortOrder: 1},
		{Slug: "apartments-rent", NameEn: "Apartments for Rent", NameAr: "شقق للإيجار", Parent: "real-estate", SortOrder: 2},
		{Slug: "villas-sale", NameEn: "Villas for Sale", NameAr: "فيلات للبيع", Parent: "real-estate", SortOrder: 3},
		{Slug: "commercial", NameEn: "Commercial Property", NameAr: "عقارات تجارية", Parent: "real-estate", SortOrder: 4},
		{Slug: "land", NameEn: "Land & Plots", NameAr: "أراضي", Parent: "real-estate", SortOrder: 5},
		{Slug: "rooms-rent", NameEn: "Rooms for Rent", NameAr: "غرف للإيجار", Parent: "real-estate", SortOrder: 6},

		// 6. Jobs & Services
		{Slug: "jobs-services", NameEn: "Jobs & Services", NameAr: "وظائف وخدمات", Color: "#735200", Icon: "💼", SortOrder: 6},
		{Slug: "jobs", NameEn: "Job Listings", NameAr: "وظائف", Parent: "jobs-services", SortOrder: 1},
		{Slug: "freelance", NameEn: "Freelance Services", NameAr: "خدمات مستقلة", Parent: "jobs-services", SortOrder: 2},
		{Slug: "home-services", NameEn: "Home Services", NameAr: "خدمات منزلية", Parent: "jobs-services", SortOrder: 3},
		{Slug: "tutoring", NameEn: "Tutoring & Lessons", NameAr: "دروس خصوصية", Parent: "jobs-services", SortOrder: 4},
		{Slug: "health-beauty-svc", NameEn: "Health & Beauty", NameAr: "صحة وجمال", Parent: "jobs-services", SortOrder: 5},
		{Slug: "events", NameEn: "Events & Photography", NameAr: "فعاليات وتصوير", Parent: "jobs-services", SortOrder: 6},

		// 7. Sports & Outdoors
		{Slug: "sports", NameEn: "Sporting Goods", NameAr: "رياضة وهواء طلق", Color: "#86B817", Icon: "⚽", SortOrder: 7},
		{Slug: "exercise", NameEn: "Exercise & Fitness", NameAr: "لياقة بدنية", Parent: "sports", SortOrder: 1},
		{Slug: "football", NameEn: "Team Sports", NameAr: "رياضات جماعية", Parent: "sports", SortOrder: 2},
		{Slug: "water-sports", NameEn: "Water Sports", NameAr: "رياضات مائية", Parent: "sports", SortOrder: 3},
		{Slug: "cycling", NameEn: "Cycling", NameAr: "دراجات", Parent: "sports", SortOrder: 4},
		{Slug: "camping", NameEn: "Camping & Hiking", NameAr: "تخييم وتسلق", Parent: "sports", SortOrder: 5},
		{Slug: "martial-arts", NameEn: "Martial Arts", NameAr: "فنون قتالية", Parent: "sports", SortOrder: 6},

		// 8. Toys & Baby
		{Slug: "toys-baby", NameEn: "Toys & Baby", NameAr: "ألعاب وأطفال", Color: "#E53238", Icon: "🧸", SortOrder: 8},
		{Slug: "toys", NameEn: "Toys", NameAr: "ألعاب", Parent: "toys-baby", SortOrder: 1},
		{Slug: "baby-gear", NameEn: "Baby Gear", NameAr: "مستلزمات أطفال", Parent: "toys-baby", SortOrder: 2},
		{Slug: "educational", NameEn: "Educational Toys", NameAr: "ألعاب تعليمية", Parent: "toys-baby", SortOrder: 3},
		{Slug: "outdoor-play", NameEn: "Outdoor Toys", NameAr: "ألعاب خارجية", Parent: "toys-baby", SortOrder: 4},
		{Slug: "video-games", NameEn: "Video Games", NameAr: "ألعاب إلكترونية", Parent: "toys-baby", SortOrder: 5},

		// 9. Business & Industrial
		{Slug: "business", NameEn: "Business & Industrial", NameAr: "أعمال وصناعة", Color: "#333333", Icon: "🏭", SortOrder: 9},
		{Slug: "office", NameEn: "Office Supplies", NameAr: "مستلزمات مكتبية", Parent: "business", SortOrder: 1},
		{Slug: "industrial", NameEn: "Industrial Equipment", NameAr: "معدات صناعية", Parent: "business", SortOrder: 2},
		{Slug: "wholesale", NameEn: "Wholesale Lots", NameAr: "بيع بالجملة", Parent: "business", SortOrder: 3},
		{Slug: "printing", NameEn: "Printing & Signage", NameAr: "طباعة وإعلانات", Parent: "business", SortOrder: 4},
		{Slug: "food-bev", NameEn: "Food & Beverage", NameAr: "أغذية ومشروبات", Parent: "business", SortOrder: 5},

		// 10. Collectibles & Art
		{Slug: "collectibles", NameEn: "Collectibles & Art", NameAr: "مقتنيات وفنون", Color: "#735200", Icon: "🏺", SortOrder: 10},
		{Slug: "antiques", NameEn: "Antiques", NameAr: "تحف وأنتيكات", Parent: "collectibles", SortOrder: 1},
		{Slug: "art", NameEn: "Art", NameAr: "فن وتشكيل", Parent: "collectibles", SortOrder: 2},
		{Slug: "coins", NameEn: "Coins & Paper Money", NameAr: "عملات نادرة", Parent: "collectibles", SortOrder: 3},
		{Slug: "stamps", NameEn: "Stamps", NameAr: "طوابع بريدية", Parent: "collectibles", SortOrder: 4},
		{Slug: "comics", NameEn: "Comics & Books", NameAr: "كتب ومجلات", Parent: "collectibles", SortOrder: 5},

		// 11. Health & Beauty
		{Slug: "health-beauty-main", NameEn: "Health & Beauty", NameAr: "صحة وجمال", Color: "#F5AF02", Icon: "💄", SortOrder: 11},
		{Slug: "skincare", NameEn: "Skin Care", NameAr: "العناية بالبشرة", Parent: "health-beauty-main", SortOrder: 1},
		{Slug: "haircare", NameEn: "Hair Care", NameAr: "العناية بالشعر", Parent: "health-beauty-main", SortOrder: 2},
		{Slug: "makeup", NameEn: "Makeup", NameAr: "مكياج", Parent: "health-beauty-main", SortOrder: 3},
		{Slug: "vitamins", NameEn: "Vitamins & Supplements", NameAr: "فيتامينات", Parent: "health-beauty-main", SortOrder: 4},
		{Slug: "medical", NameEn: "Medical & Mobility", NameAr: "معدات طبية", Parent: "health-beauty-main", SortOrder: 5},

		// 12. Pet Supplies
		{Slug: "pets", NameEn: "Pet Supplies", NameAr: "مستلزمات الحيوانات", Color: "#86B817", Icon: "🐾", SortOrder: 12},
		{Slug: "dog", NameEn: "Dog Supplies", NameAr: "مستلزمات الكلاب", Parent: "pets", SortOrder: 1},
		{Slug: "cat", NameEn: "Cat Supplies", NameAr: "مستلزمات القطط", Parent: "pets", SortOrder: 2},
		{Slug: "fish", NameEn: "Fish & Aquatic", NameAr: "أسماك وأحياء مائية", Parent: "pets", SortOrder: 3},
		{Slug: "birds", NameEn: "Bird Supplies", NameAr: "مستلزمات الطيور", Parent: "pets", SortOrder: 4},
	}

	// ════════════════════════════════════════════════════════════════════════════
	// Custom Fields per Category
	// ════════════════════════════════════════════════════════════════════════════
	fields := []seedField{
		// Smartphones
		{CategorySlug: "smartphones", Name: "brand", LabelEn: "Brand", LabelAr: "الماركة", FieldType: "select", Options: []string{"Apple", "Samsung", "Huawei", "Xiaomi", "Oppo", "Vivo", "OnePlus", "Other"}, IsRequired: true, SortOrder: 1},
		{CategorySlug: "smartphones", Name: "model", LabelEn: "Model", LabelAr: "الموديل", FieldType: "text", IsRequired: true, SortOrder: 2},
		{CategorySlug: "smartphones", Name: "storage", LabelEn: "Storage", LabelAr: "السعة", FieldType: "select", Options: []string{"16GB", "32GB", "64GB", "128GB", "256GB", "512GB", "1TB"}, SortOrder: 3},
		{CategorySlug: "smartphones", Name: "ram", LabelEn: "RAM", LabelAr: "الرام", FieldType: "select", Options: []string{"2GB", "3GB", "4GB", "6GB", "8GB", "12GB", "16GB"}, SortOrder: 4},
		{CategorySlug: "smartphones", Name: "condition", LabelEn: "Condition", LabelAr: "الحالة", FieldType: "select", Options: []string{"Brand New", "Like New", "Good", "Fair", "For Parts"}, IsRequired: true, SortOrder: 5},
		{CategorySlug: "smartphones", Name: "color", LabelEn: "Color", LabelAr: "اللون", FieldType: "text", SortOrder: 6},
		{CategorySlug: "smartphones", Name: "warranty", LabelEn: "Warranty", LabelAr: "الضمان", FieldType: "select", Options: []string{"No Warranty", "3 Months", "6 Months", "1 Year", "2 Years"}, SortOrder: 7},

		// Cars & Trucks
		{CategorySlug: "cars-trucks", Name: "make", LabelEn: "Make", LabelAr: "الشركة المصنعة", FieldType: "select", Options: []string{"Toyota", "Honda", "BMW", "Mercedes", "Hyundai", "Kia", "Ford", "Chevrolet", "Nissan", "Volkswagen", "Audi", "Jeep", "Other"}, IsRequired: true, SortOrder: 1},
		{CategorySlug: "cars-trucks", Name: "model", LabelEn: "Model", LabelAr: "الموديل", FieldType: "text", IsRequired: true, SortOrder: 2},
		{CategorySlug: "cars-trucks", Name: "year", LabelEn: "Year", LabelAr: "سنة الصنع", FieldType: "number", IsRequired: true, SortOrder: 3},
		{CategorySlug: "cars-trucks", Name: "mileage", LabelEn: "Mileage (km)", LabelAr: "المسافة (كم)", FieldType: "number", SortOrder: 4},
		{CategorySlug: "cars-trucks", Name: "fuel_type", LabelEn: "Fuel Type", LabelAr: "نوع الوقود", FieldType: "select", Options: []string{"Petrol", "Diesel", "Hybrid", "Electric", "Natural Gas"}, SortOrder: 5},
		{CategorySlug: "cars-trucks", Name: "transmission", LabelEn: "Transmission", LabelAr: "ناقل الحركة", FieldType: "select", Options: []string{"Automatic", "Manual"}, SortOrder: 6},
		{CategorySlug: "cars-trucks", Name: "color", LabelEn: "Color", LabelAr: "اللون", FieldType: "text", SortOrder: 7},
		{CategorySlug: "cars-trucks", Name: "body_type", LabelEn: "Body Type", LabelAr: "نوع الهيكل", FieldType: "select", Options: []string{"Sedan", "SUV", "Coupe", "Pickup", "Hatchback", "Convertible", "Van", "Minivan"}, SortOrder: 8},
		{CategorySlug: "cars-trucks", Name: "condition", LabelEn: "Condition", LabelAr: "الحالة", FieldType: "select", Options: []string{"New", "Used - Excellent", "Used - Good", "Used - Fair", "For Parts"}, IsRequired: true, SortOrder: 9},

		// Apartments for Sale
		{CategorySlug: "apartments-sale", Name: "area_sqm", LabelEn: "Area (sqm)", LabelAr: "المساحة (م²)", FieldType: "number", IsRequired: true, SortOrder: 1},
		{CategorySlug: "apartments-sale", Name: "bedrooms", LabelEn: "Bedrooms", LabelAr: "غرف النوم", FieldType: "select", Options: []string{"Studio", "1", "2", "3", "4", "5", "6+"}, IsRequired: true, SortOrder: 2},
		{CategorySlug: "apartments-sale", Name: "bathrooms", LabelEn: "Bathrooms", LabelAr: "الحمامات", FieldType: "select", Options: []string{"1", "2", "3", "4+"}, SortOrder: 3},
		{CategorySlug: "apartments-sale", Name: "floor", LabelEn: "Floor", LabelAr: "الطابق", FieldType: "number", SortOrder: 4},
		{CategorySlug: "apartments-sale", Name: "total_floors", LabelEn: "Total Floors", LabelAr: "إجمالي الطوابق", FieldType: "number", SortOrder: 5},
		{CategorySlug: "apartments-sale", Name: "furnished", LabelEn: "Furnished", LabelAr: "مفروش", FieldType: "select", Options: []string{"Fully Furnished", "Semi Furnished", "Unfurnished"}, SortOrder: 6},
		{CategorySlug: "apartments-sale", Name: "finish", LabelEn: "Finishing", LabelAr: "التشطيب", FieldType: "select", Options: []string{"Super Lux", "Lux", "Semi Finished", "Core & Shell"}, SortOrder: 7},
		{CategorySlug: "apartments-sale", Name: "compound", LabelEn: "Compound", LabelAr: "الكمبوند", FieldType: "text", SortOrder: 8},
		{CategorySlug: "apartments-sale", Name: "year_built", LabelEn: "Year Built", LabelAr: "سنة البناء", FieldType: "number", SortOrder: 9},
	}

	// ════════════════════════════════════════════════════════════════════════════
	// Insert categories (resolve parent by slug)
	// ════════════════════════════════════════════════════════════════════════════
	slugMap := make(map[string]uuid.UUID, len(cats))

	// First pass: L1 categories (no parent)
	for _, c := range cats {
		if c.Parent != "" {
			continue
		}
		id := uuid.New()
		cat := Category{
			ID: id, Slug: c.Slug, NameEn: c.NameEn, NameAr: c.NameAr,
			Color: c.Color, Icon: c.Icon, IsLeaf: c.IsLeaf,
			SortOrder: c.SortOrder, IsActive: true,
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&cat).Error; err != nil {
			slog.Warn("seed category L1 failed", "slug", c.Slug, "err", err)
		}
		slugMap[c.Slug] = id
	}

	// Second pass: L2+ categories (resolve parent)
	for _, c := range cats {
		if c.Parent == "" {
			continue
		}
		parentID, ok := slugMap[c.Parent]
		if !ok {
			slog.Warn("seed category parent not found", "slug", c.Slug, "parent", c.Parent)
			continue
		}
		id := uuid.New()
		cat := Category{
			ID: id, ParentID: &parentID, Slug: c.Slug, NameEn: c.NameEn, NameAr: c.NameAr,
			Icon: c.Icon, IsLeaf: c.IsLeaf,
			SortOrder: c.SortOrder, IsActive: true,
		}
		// Inherit color from L1 parent if not set
		if c.Color == "" {
			// walk up to find L1 color
			for pSlug := c.Parent; pSlug != ""; {
				for _, pc := range cats {
					if pc.Slug == pSlug && pc.Parent == "" {
						cat.Color = pc.Color
						pSlug = ""
						break
					} else if pc.Slug == pSlug {
						pSlug = pc.Parent
						break
					}
				}
			}
		} else {
			cat.Color = c.Color
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&cat).Error; err != nil {
			slog.Warn("seed category L2+ failed", "slug", c.Slug, "err", err)
		}
		slugMap[c.Slug] = id
	}

	// ════════════════════════════════════════════════════════════════════════════
	// Insert category fields
	// ════════════════════════════════════════════════════════════════════════════
	for _, f := range fields {
		catID, ok := slugMap[f.CategorySlug]
		if !ok {
			slog.Warn("seed field category not found", "slug", f.CategorySlug)
			continue
		}
		optsJSON, _ := json.Marshal(f.Options)
		cf := CategoryField{
			ID: uuid.New(), CategoryID: catID,
			Name: f.Name, Label: f.LabelEn, LabelEn: f.LabelEn, LabelAr: f.LabelAr,
			FieldType: f.FieldType, Options: string(optsJSON),
			IsRequired: f.IsRequired, SortOrder: f.SortOrder, IsActive: true,
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&cf).Error; err != nil {
			slog.Warn("seed field failed", "name", f.Name, "err", err)
		}
	}

	slog.Info("✅ category tree seed complete", "categories", len(cats), "fields", len(fields))
}
