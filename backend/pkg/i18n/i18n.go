package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Supported languages
const (
	LangEnglish = "en"
	LangArabic  = "ar"
	LangFrench  = "fr"
	LangSpanish = "es"
	LangGerman  = "de"
	LangTurkish = "tr"
)

// RTL languages
var RTLLanguages = map[string]bool{
	"ar": true,
	"he": true,
	"fa": true,
	"ur": true,
}

// DefaultLanguage is the fallback language
const DefaultLanguage = LangEnglish

// Translator handles internationalization
type Translator struct {
	translations map[string]map[string]string // locale -> key -> value
	mu           sync.RWMutex
	defaultLang  string
}

// Global translator instance
var globalTranslator *Translator
var once sync.Once

// GetTranslator returns the global translator instance
func GetTranslator() *Translator {
	once.Do(func() {
		globalTranslator = &Translator{
			translations: make(map[string]map[string]string),
			defaultLang:  DefaultLanguage,
		}
	})
	return globalTranslator
}

// LoadTranslations loads translation files from a directory
func (t *Translator) LoadTranslations(dir string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		locale := strings.TrimSuffix(filepath.Base(file), ".json")
		
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}

		t.translations[locale] = translations
	}

	return nil
}

// LoadEmbeddedTranslations loads translations from embedded data
func (t *Translator) LoadEmbeddedTranslations() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// English translations
	t.translations["en"] = map[string]string{
		// Common
		"common.success":           "Success",
		"common.error":             "Error",
		"common.loading":           "Loading...",
		"common.save":              "Save",
		"common.cancel":            "Cancel",
		"common.delete":            "Delete",
		"common.edit":              "Edit",
		"common.view":              "View",
		"common.search":            "Search",
		"common.filter":            "Filter",
		"common.sort":              "Sort",
		"common.next":              "Next",
		"common.previous":          "Previous",
		"common.submit":            "Submit",
		"common.confirm":           "Confirm",
		"common.yes":               "Yes",
		"common.no":                "No",

		// Auth
		"auth.login":               "Login",
		"auth.logout":              "Logout",
		"auth.register":            "Register",
		"auth.forgot_password":     "Forgot Password",
		"auth.reset_password":      "Reset Password",
		"auth.email":               "Email",
		"auth.password":            "Password",
		"auth.confirm_password":    "Confirm Password",
		"auth.invalid_credentials": "Invalid email or password",
		"auth.account_created":     "Account created successfully",
		"auth.password_reset_sent": "Password reset link sent to your email",

		// Auctions
		"auction.title":            "Auction",
		"auction.place_bid":        "Place Bid",
		"auction.current_bid":      "Current Bid",
		"auction.starting_price":   "Starting Price",
		"auction.buy_now":          "Buy Now",
		"auction.time_left":        "Time Left",
		"auction.ended":            "Auction Ended",
		"auction.won":              "You won this auction!",
		"auction.outbid":           "You have been outbid",
		"auction.bid_placed":       "Bid placed successfully",
		"auction.bid_too_low":      "Bid must be higher than current bid",
		"auction.extended":         "Auction extended due to last-minute bid",

		// Wallet
		"wallet.balance":           "Balance",
		"wallet.deposit":           "Deposit",
		"wallet.withdraw":          "Withdraw",
		"wallet.transaction":       "Transaction",
		"wallet.insufficient":      "Insufficient balance",

		// Disputes
		"dispute.open":             "Open Dispute",
		"dispute.resolved":         "Dispute Resolved",
		"dispute.escalated":        "Dispute Escalated",
		"dispute.awaiting":         "Awaiting Response",

		// Loyalty
		"loyalty.points":           "Points",
		"loyalty.tier":             "Tier",
		"loyalty.redeem":           "Redeem",
		"loyalty.daily_bonus":      "Daily Bonus",
		"loyalty.referral":         "Referral",

		// Notifications
		"notification.new_bid":     "New bid on your auction",
		"notification.auction_won": "Congratulations! You won the auction",
		"notification.outbid":      "Someone outbid you",
		"notification.auction_end": "Auction ending soon",

		// Errors
		"error.not_found":          "Not found",
		"error.unauthorized":       "Unauthorized",
		"error.forbidden":          "Access denied",
		"error.validation":         "Validation error",
		"error.internal":           "Internal server error",
		"error.rate_limit":         "Too many requests",
	}

	// Arabic translations (RTL)
	t.translations["ar"] = map[string]string{
		// Common
		"common.success":           "نجاح",
		"common.error":             "خطأ",
		"common.loading":           "جاري التحميل...",
		"common.save":              "حفظ",
		"common.cancel":            "إلغاء",
		"common.delete":            "حذف",
		"common.edit":              "تعديل",
		"common.view":              "عرض",
		"common.search":            "بحث",
		"common.filter":            "تصفية",
		"common.sort":              "ترتيب",
		"common.next":              "التالي",
		"common.previous":          "السابق",
		"common.submit":            "إرسال",
		"common.confirm":           "تأكيد",
		"common.yes":               "نعم",
		"common.no":                "لا",

		// Auth
		"auth.login":               "تسجيل الدخول",
		"auth.logout":              "تسجيل الخروج",
		"auth.register":            "إنشاء حساب",
		"auth.forgot_password":     "نسيت كلمة المرور",
		"auth.reset_password":      "إعادة تعيين كلمة المرور",
		"auth.email":               "البريد الإلكتروني",
		"auth.password":            "كلمة المرور",
		"auth.confirm_password":    "تأكيد كلمة المرور",
		"auth.invalid_credentials": "البريد الإلكتروني أو كلمة المرور غير صحيحة",
		"auth.account_created":     "تم إنشاء الحساب بنجاح",
		"auth.password_reset_sent": "تم إرسال رابط إعادة تعيين كلمة المرور",

		// Auctions
		"auction.title":            "المزاد",
		"auction.place_bid":        "تقديم عرض",
		"auction.current_bid":      "العرض الحالي",
		"auction.starting_price":   "السعر الابتدائي",
		"auction.buy_now":          "اشتر الآن",
		"auction.time_left":        "الوقت المتبقي",
		"auction.ended":            "انتهى المزاد",
		"auction.won":              "مبروك! فزت بالمزاد",
		"auction.outbid":           "تم تجاوز عرضك",
		"auction.bid_placed":       "تم تقديم العرض بنجاح",
		"auction.bid_too_low":      "يجب أن يكون العرض أعلى من العرض الحالي",
		"auction.extended":         "تم تمديد المزاد بسبب عرض في اللحظة الأخيرة",

		// Wallet
		"wallet.balance":           "الرصيد",
		"wallet.deposit":           "إيداع",
		"wallet.withdraw":          "سحب",
		"wallet.transaction":       "معاملة",
		"wallet.insufficient":      "رصيد غير كافٍ",

		// Disputes
		"dispute.open":             "فتح نزاع",
		"dispute.resolved":         "تم حل النزاع",
		"dispute.escalated":        "تم تصعيد النزاع",
		"dispute.awaiting":         "في انتظار الرد",

		// Loyalty
		"loyalty.points":           "النقاط",
		"loyalty.tier":             "المستوى",
		"loyalty.redeem":           "استبدال",
		"loyalty.daily_bonus":      "المكافأة اليومية",
		"loyalty.referral":         "الإحالة",

		// Notifications
		"notification.new_bid":     "عرض جديد على مزادك",
		"notification.auction_won": "مبروك! فزت بالمزاد",
		"notification.outbid":      "شخص ما تجاوز عرضك",
		"notification.auction_end": "المزاد ينتهي قريباً",

		// Errors
		"error.not_found":          "غير موجود",
		"error.unauthorized":       "غير مصرح",
		"error.forbidden":          "الوصول مرفوض",
		"error.validation":         "خطأ في التحقق",
		"error.internal":           "خطأ في الخادم",
		"error.rate_limit":         "طلبات كثيرة جداً",
	}

	// French translations
	t.translations["fr"] = map[string]string{
		"common.success":           "Succès",
		"common.error":             "Erreur",
		"common.loading":           "Chargement...",
		"common.save":              "Enregistrer",
		"common.cancel":            "Annuler",
		"auth.login":               "Connexion",
		"auth.logout":              "Déconnexion",
		"auth.register":            "S'inscrire",
		"auction.place_bid":        "Placer une enchère",
		"auction.current_bid":      "Enchère actuelle",
		"auction.buy_now":          "Acheter maintenant",
	}

	// Spanish translations
	t.translations["es"] = map[string]string{
		"common.success":           "Éxito",
		"common.error":             "Error",
		"common.loading":           "Cargando...",
		"common.save":              "Guardar",
		"common.cancel":            "Cancelar",
		"auth.login":               "Iniciar sesión",
		"auth.logout":              "Cerrar sesión",
		"auth.register":            "Registrarse",
		"auction.place_bid":        "Hacer oferta",
		"auction.current_bid":      "Oferta actual",
		"auction.buy_now":          "Comprar ahora",
	}
}

// T translates a key to the specified locale
func (t *Translator) T(locale, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try requested locale
	if translations, ok := t.translations[locale]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}

	// Fallback to default language
	if translations, ok := t.translations[t.defaultLang]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}

	// Return key if no translation found
	return key
}

// TF translates with format arguments
func (t *Translator) TF(locale, key string, args ...interface{}) string {
	return fmt.Sprintf(t.T(locale, key), args...)
}

// IsRTL checks if a language is RTL
func IsRTL(locale string) bool {
	return RTLLanguages[locale]
}

// GetDirection returns text direction for a locale
func GetDirection(locale string) string {
	if IsRTL(locale) {
		return "rtl"
	}
	return "ltr"
}

// Middleware adds i18n support to Gin context
func Middleware() gin.HandlerFunc {
	translator := GetTranslator()
	translator.LoadEmbeddedTranslations()

	return func(c *gin.Context) {
		// Get locale from header, query, or cookie
		locale := c.GetHeader("Accept-Language")
		if locale == "" {
			locale = c.Query("lang")
		}
		if locale == "" {
			locale, _ = c.Cookie("locale")
		}
		if locale == "" {
			locale = DefaultLanguage
		}

		// Extract primary language (e.g., "en-US" -> "en")
		if idx := strings.Index(locale, "-"); idx > 0 {
			locale = locale[:idx]
		}
		if idx := strings.Index(locale, "_"); idx > 0 {
			locale = locale[:idx]
		}

		// Validate locale
		if _, ok := translator.translations[locale]; !ok {
			locale = DefaultLanguage
		}

		// Set in context
		c.Set("locale", locale)
		c.Set("direction", GetDirection(locale))
		c.Set("is_rtl", IsRTL(locale))

		// Helper function for templates
		c.Set("t", func(key string) string {
			return translator.T(locale, key)
		})

		c.Next()
	}
}

// GetLocale returns the current locale from context
func GetLocale(c *gin.Context) string {
	if locale, ok := c.Get("locale"); ok {
		return locale.(string)
	}
	return DefaultLanguage
}

// T is a shortcut for translating in handlers
func T(c *gin.Context, key string) string {
	return GetTranslator().T(GetLocale(c), key)
}

// TF is a shortcut for translating with format in handlers
func TF(c *gin.Context, key string, args ...interface{}) string {
	return GetTranslator().TF(GetLocale(c), key, args...)
}
