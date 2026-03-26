package listings

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedCategories(db *gorm.DB) {
	var count int64
	db.Model(&Category{}).Count(&count)
	if count > 0 {
		return
	}

	cats := []Category{
		{ID: uuid.New(), NameEn: "Vehicles", NameAr: "السيارات", Slug: "vehicles", Icon: "🚗", SortOrder: 1},
		{ID: uuid.New(), NameEn: "Real Estate", NameAr: "العقارات", Slug: "real-estate", Icon: "🏠", SortOrder: 2},
		{ID: uuid.New(), NameEn: "Electronics", NameAr: "الإلكترونيات", Slug: "electronics", Icon: "📱", SortOrder: 3},
		{ID: uuid.New(), NameEn: "Furniture", NameAr: "الأثاث", Slug: "furniture", Icon: "🛋️", SortOrder: 4},
		{ID: uuid.New(), NameEn: "Clothing", NameAr: "الملابس", Slug: "clothing", Icon: "👕", SortOrder: 5},
		{ID: uuid.New(), NameEn: "Jobs", NameAr: "الوظائف", Slug: "jobs", Icon: "💼", SortOrder: 6},
		{ID: uuid.New(), NameEn: "Services", NameAr: "الخدمات", Slug: "services", Icon: "🔧", SortOrder: 7},
		{ID: uuid.New(), NameEn: "Animals & Pets", NameAr: "الحيوانات", Slug: "animals-pets", Icon: "🐾", SortOrder: 8},
		{ID: uuid.New(), NameEn: "Sports & Hobbies", NameAr: "الرياضة", Slug: "sports-hobbies", Icon: "⚽", SortOrder: 9},
		{ID: uuid.New(), NameEn: "Kids & Baby", NameAr: "الأطفال", Slug: "kids-baby", Icon: "🧸", SortOrder: 10},
	}
	db.Create(&cats)
}
