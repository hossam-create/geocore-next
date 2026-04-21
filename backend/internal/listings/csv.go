package listings

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var csvHeader = []string{
	"id", "title", "description", "price", "currency", "condition",
	"status", "country", "city", "category_slug", "custom_fields", "created_at",
}

// ExportCSV exports all listings for the authenticated user as CSV.
func (h *Handler) ExportCSV(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var listings []Listing
	h.db.Preload("Category").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&listings)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="my-listings-%s.csv"`, time.Now().Format("2006-01-02")))

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	w.Write(csvHeader)

	for _, l := range listings {
		catSlug := ""
		if l.Category != nil {
			catSlug = l.Category.Slug
		}
		priceStr := ""
		if l.Price != nil {
			priceStr = strconv.FormatFloat(*l.Price, 'f', 2, 64)
		}
		w.Write([]string{
			l.ID.String(),
			l.Title,
			l.Description,
			priceStr,
			l.Currency,
			l.Condition,
			l.Status,
			l.Country,
			l.City,
			catSlug,
			l.CustomFields,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
}

// ExportTemplate returns an empty CSV with headers only.
func (h *Handler) ExportTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="listings-template.csv"`)

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	// Template header (without id and created_at — those are auto-generated)
	w.Write([]string{
		"title", "description", "price", "currency", "condition",
		"country", "city", "category_slug", "custom_fields",
	})
	// Example row
	w.Write([]string{
		"Sample Laptop", "Great condition laptop", "499.99", "USD", "good",
		"USA", "New York", "electronics", `{"brand":"Dell"}`,
	})
}

// ImportCSV imports listings from a CSV file upload.
func (h *Handler) ImportCSV(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "CSV file is required (field: file)")
		return
	}
	defer file.Close()

	r := csv.NewReader(file)

	// Read and validate header
	header, err := r.Read()
	if err != nil {
		response.BadRequest(c, "Could not read CSV header")
		return
	}

	colIndex := map[string]int{}
	for i, col := range header {
		colIndex[strings.TrimSpace(strings.ToLower(col))] = i
	}

	requiredCols := []string{"title", "description", "country", "city", "category_slug"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			response.BadRequest(c, fmt.Sprintf("Missing required column: %s", col))
			return
		}
	}

	type importError struct {
		Row    int    `json:"row"`
		Reason string `json:"reason"`
	}

	var (
		success int
		failed  int
		errors  []importError
	)

	maxRows := 500
	rowNum := 1
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		if err != nil {
			failed++
			errors = append(errors, importError{Row: rowNum, Reason: "malformed CSV row"})
			continue
		}
		if rowNum > maxRows+1 {
			errors = append(errors, importError{Row: rowNum, Reason: "max 500 rows exceeded"})
			break
		}

		getCol := func(name string) string {
			if idx, ok := colIndex[name]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		title := getCol("title")
		desc := getCol("description")
		country := getCol("country")
		city := getCol("city")
		catSlug := getCol("category_slug")

		if title == "" || desc == "" || country == "" || city == "" || catSlug == "" {
			failed++
			errors = append(errors, importError{Row: rowNum, Reason: "missing required field (title/description/country/city/category_slug)"})
			continue
		}

		// Resolve category
		var cat Category
		if err := h.db.Where("slug = ?", catSlug).First(&cat).Error; err != nil {
			failed++
			errors = append(errors, importError{Row: rowNum, Reason: fmt.Sprintf("unknown category: %s", catSlug)})
			continue
		}

		var price *float64
		if p := getCol("price"); p != "" {
			pf, err := strconv.ParseFloat(p, 64)
			if err != nil {
				failed++
				errors = append(errors, importError{Row: rowNum, Reason: "invalid price"})
				continue
			}
			price = &pf
		}

		condition := getCol("condition")
		if condition == "" {
			condition = "good"
		}

		currency := getCol("currency")
		if currency == "" {
			currency = "USD"
		}

		cf := getCol("custom_fields")
		if cf == "" {
			cf = "{}"
		}
		// Validate JSON
		var cfParsed any
		if json.Unmarshal([]byte(cf), &cfParsed) != nil {
			cf = "{}"
		}

		expires := time.Now().AddDate(0, 2, 0)
		listing := Listing{
			ID:           uuid.New(),
			UserID:       userID,
			CategoryID:   cat.ID,
			Title:        title,
			Description:  desc,
			Price:        price,
			Currency:     currency,
			Condition:    condition,
			Country:      country,
			City:         city,
			Status:       "active",
			CustomFields: cf,
			ExpiresAt:    &expires,
		}

		if err := h.db.Create(&listing).Error; err != nil {
			failed++
			errors = append(errors, importError{Row: rowNum, Reason: err.Error()})
			continue
		}
		success++
	}

	response.OK(c, gin.H{
		"success": success,
		"failed":  failed,
		"errors":  errors,
	})
}
