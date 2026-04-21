package country

import (
	"log/slog"

	"gorm.io/gorm"
)

// SeedCountryConfigs inserts default country configurations for GCC + Egypt markets.
// Called once on startup; existing records are updated (Upsert via Save).
func SeedCountryConfigs(db *gorm.DB) {
	db.AutoMigrate(&CountryConfig{}, &CountryOverride{})

	configs := []CountryConfig{
		{
			Code: "EG", NameEn: "Egypt", NameAr: "مصر",
			Currency: "EGP", CurrencyAr: "جنيه", CurrencySymbol: "E£",
			TaxRate: 0.14, TaxLabel: "VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 5000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "cash_on_delivery", "wallet", "paymob", "bnpl"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: true, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: true, EnableRealEstate: true,
			DefaultShippingCents: 5000, FreeShippingThresholdCents: 100000,
			ShippingZones:        []string{"cairo", "alexandria", "giza", "other"},
			Locale: "ar-EG", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: false, MinAge: 18, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "SA", NameEn: "Saudi Arabia", NameAr: "المملكة العربية السعودية",
			Currency: "SAR", CurrencyAr: "ريال", CurrencySymbol: "﷼",
			TaxRate: 0.15, TaxLabel: "VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 10000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "mada", "stc_pay", "bnpl", "tabby", "tamara"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: true, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: true, EnableRealEstate: true,
			DefaultShippingCents: 15000, FreeShippingThresholdCents: 200000,
			ShippingZones:        []string{"riyadh", "jeddah", "dammam", "other"},
			Locale: "ar-SA", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: true, MinAge: 18, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "AE", NameEn: "United Arab Emirates", NameAr: "الإمارات العربية المتحدة",
			Currency: "AED", CurrencyAr: "درهم", CurrencySymbol: "د.إ",
			TaxRate: 0.05, TaxLabel: "VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 1000000, KYCTier2LimitCents: 20000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "apple_pay", "tabby", "tamara", "crypto"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: true, EnableP2P: true,
			EnableCrypto: true, EnableCrowdship: true, EnableWholesale: true, EnableRealEstate: true,
			DefaultShippingCents: 10000, FreeShippingThresholdCents: 150000,
			ShippingZones:        []string{"dubai", "abu_dhabi", "sharjah", "other"},
			Locale: "ar-AE", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: true, MinAge: 21, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "KW", NameEn: "Kuwait", NameAr: "الكويت",
			Currency: "KWD", CurrencyAr: "دينار", CurrencySymbol: "د.ك",
			TaxRate: 0.00, TaxLabel: "No VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 10000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "knet", "cash_on_delivery"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: false, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: false, EnableRealEstate: true,
			DefaultShippingCents: 10000, FreeShippingThresholdCents: 200000,
			ShippingZones:        []string{"kuwait_city", "hawalli", "other"},
			Locale: "ar-KW", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: false, MinAge: 21, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "BH", NameEn: "Bahrain", NameAr: "البحرين",
			Currency: "BHD", CurrencyAr: "دينار", CurrencySymbol: "د.ب",
			TaxRate: 0.10, TaxLabel: "VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 5000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "benefit", "cash_on_delivery"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: false, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: false, EnableRealEstate: true,
			DefaultShippingCents: 8000, FreeShippingThresholdCents: 150000,
			ShippingZones:        []string{"manama", "muharraq", "other"},
			Locale: "ar-BH", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: false, MinAge: 21, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "QA", NameEn: "Qatar", NameAr: "قطر",
			Currency: "QAR", CurrencyAr: "ريال", CurrencySymbol: "ر.ق",
			TaxRate: 0.00, TaxLabel: "No VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 10000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "qpay", "cash_on_delivery"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: false, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: false, EnableRealEstate: true,
			DefaultShippingCents: 10000, FreeShippingThresholdCents: 200000,
			ShippingZones:        []string{"doha", "al_wakrah", "other"},
			Locale: "ar-QA", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: true, MinAge: 21, MaxReturnDays: 14,
			IsActive: true,
		},
		{
			Code: "OM", NameEn: "Oman", NameAr: "عمان",
			Currency: "OMR", CurrencyAr: "ريال", CurrencySymbol: "ر.ع",
			TaxRate: 0.05, TaxLabel: "VAT", TaxInclusive: true,
			ServiceFeeRate: 0.05, WithholdingRate: 0,
			KYCTier1LimitCents: 500000, KYCTier2LimitCents: 5000000, KYCTier3LimitCents: 0,
			MaxListingPriceCents: 0,
			PaymentMethods:       []string{"card", "wallet", "cash_on_delivery"},
			EnableAuctions: true, EnableLive: true, EnableBNPL: false, EnableP2P: false,
			EnableCrypto: false, EnableCrowdship: true, EnableWholesale: false, EnableRealEstate: true,
			DefaultShippingCents: 8000, FreeShippingThresholdCents: 150000,
			ShippingZones:        []string{"muscat", "salalah", "other"},
			Locale: "ar-OM", DateFormat: "DD/MM/YYYY", NumberFormat: "1,234.56",
			RequireNationalID: true, RequireAddressProof: false, MinAge: 21, MaxReturnDays: 14,
			IsActive: true,
		},
	}

	for _, cfg := range configs {
		result := db.Where("code = ?", cfg.Code).FirstOrCreate(&cfg)
		if result.Error != nil {
			slog.Error("country seed: failed", "code", cfg.Code, "error", result.Error.Error())
		} else if result.RowsAffected > 0 {
			slog.Info("country seed: created", "code", cfg.Code)
		}
	}
}
