package validator

  import (
  	"fmt"
  	"regexp"
  	"strings"
  	"unicode"
  )

  // ════════════════════════════════════════════════════════════════════════════
  // Compiled regular expressions (initialised once at startup)
  // ════════════════════════════════════════════════════════════════════════════

  var (
  	// RFC 5322-compatible simplified email pattern
  	emailRegex = regexp.MustCompile(
  		`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
  	)

  	// E.164 international phone format: +971501234567
  	e164Regex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

  	// Basic HTTP/HTTPS URL
  	urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

  	// Control characters (except \t, \n, \r which are legitimate whitespace)
  	controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)

  	// HTML tags pattern (for stripping markup)
  	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
  )

  // ════════════════════════════════════════════════════════════════════════════
  // FieldError — a single field-level validation failure
  // ════════════════════════════════════════════════════════════════════════════

  type FieldError struct {
  	Field   string `json:"field"`
  	Message string `json:"message"`
  }

  func (e FieldError) Error() string {
  	return fmt.Sprintf("%s: %s", e.Field, e.Message)
  }

  // ════════════════════════════════════════════════════════════════════════════
  // ValidationErrors — collection returned when validation fails
  // ════════════════════════════════════════════════════════════════════════════

  type ValidationErrors []FieldError

  func (ve ValidationErrors) Error() string {
  	msgs := make([]string, len(ve))
  	for i, e := range ve {
  		msgs[i] = e.Error()
  	}
  	return strings.Join(msgs, "; ")
  }

  // Any reports whether there are validation errors.
  func (ve ValidationErrors) Any() bool { return len(ve) > 0 }

  // ════════════════════════════════════════════════════════════════════════════
  // Request types — validated input structs
  // ════════════════════════════════════════════════════════════════════════════

  // RegisterRequest is the payload for POST /auth/register.
  type RegisterRequest struct {
  	Email    string `json:"email"    binding:"required"`
  	Password string `json:"password" binding:"required"`
  	Name     string `json:"name"     binding:"required"`
  	Phone    string `json:"phone"`
  }

  func (r *RegisterRequest) Validate() ValidationErrors {
  	var errs ValidationErrors

  	// Sanitise inputs first
  	r.Email = SanitizeString(strings.ToLower(r.Email))
  	r.Name  = SanitizeName(r.Name)
  	r.Phone = SanitizeString(r.Phone)

  	// Email
  	switch {
  	case r.Email == "":
  		errs = append(errs, FieldError{"email", "email is required"})
  	case len(r.Email) > 255:
  		errs = append(errs, FieldError{"email", "email must not exceed 255 characters"})
  	case !emailRegex.MatchString(r.Email):
  		errs = append(errs, FieldError{"email", "must be a valid email address"})
  	}

  	// Password
  	if pwErrs := ValidatePassword("password", r.Password); len(pwErrs) > 0 {
  		errs = append(errs, pwErrs...)
  	}

  	// Name
  	switch {
  	case r.Name == "":
  		errs = append(errs, FieldError{"name", "name is required"})
  	case len(r.Name) < 2:
  		errs = append(errs, FieldError{"name", "name must be at least 2 characters"})
  	case len(r.Name) > 100:
  		errs = append(errs, FieldError{"name", "name must not exceed 100 characters"})
  	}

  	// Phone (optional)
  	if r.Phone != "" && !e164Regex.MatchString(r.Phone) {
  		errs = append(errs, FieldError{"phone",
  			"phone must be in E.164 format (e.g. +971501234567)"})
  	}

  	return errs
  }

  // LoginRequest is the payload for POST /auth/login.
  type LoginRequest struct {
  	Email    string `json:"email"    binding:"required"`
  	Password string `json:"password" binding:"required"`
  }

  func (r *LoginRequest) Validate() ValidationErrors {
  	var errs ValidationErrors

  	r.Email = SanitizeString(strings.ToLower(r.Email))

  	if r.Email == "" {
  		errs = append(errs, FieldError{"email", "email is required"})
  	} else if !emailRegex.MatchString(r.Email) {
  		errs = append(errs, FieldError{"email", "must be a valid email address"})
  	}

  	if r.Password == "" {
  		errs = append(errs, FieldError{"password", "password is required"})
  	}

  	return errs
  }

  // Location is embedded in listing and auction requests.
  type Location struct {
  	Latitude  float64 `json:"latitude"`
  	Longitude float64 `json:"longitude"`
  	Address   string  `json:"address"`
  }

  func (l *Location) Validate() ValidationErrors {
  	var errs ValidationErrors
  	l.Address = SanitizeString(l.Address)

  	if l.Latitude < -90 || l.Latitude > 90 {
  		errs = append(errs, FieldError{"location.latitude",
  			"latitude must be between -90 and 90"})
  	}
  	if l.Longitude < -180 || l.Longitude > 180 {
  		errs = append(errs, FieldError{"location.longitude",
  			"longitude must be between -180 and 180"})
  	}
  	switch {
  	case l.Address == "":
  		errs = append(errs, FieldError{"location.address", "address is required"})
  	case len(l.Address) > 500:
  		errs = append(errs, FieldError{"location.address",
  			"address must not exceed 500 characters"})
  	}
  	return errs
  }

  // CreateListingRequest is the payload for POST /listings.
  type CreateListingRequest struct {
  	Title       string    `json:"title"       binding:"required"`
  	Description string    `json:"description" binding:"required"`
  	Price       float64   `json:"price"       binding:"required"`
  	CategoryID  int64     `json:"category_id" binding:"required"`
  	Condition   string    `json:"condition"`
  	Location    Location  `json:"location"`
  	Images      []string  `json:"images"`
  }

  func (r *CreateListingRequest) Validate() ValidationErrors {
  	var errs ValidationErrors

  	r.Title       = SanitizeString(r.Title)
  	r.Description = SanitizeRichText(r.Description)
  	r.Condition   = SanitizeString(r.Condition)

  	// Title
  	switch {
  	case r.Title == "":
  		errs = append(errs, FieldError{"title", "title is required"})
  	case len(r.Title) < 5:
  		errs = append(errs, FieldError{"title", "title must be at least 5 characters"})
  	case len(r.Title) > 200:
  		errs = append(errs, FieldError{"title", "title must not exceed 200 characters"})
  	}

  	// Description
  	switch {
  	case r.Description == "":
  		errs = append(errs, FieldError{"description", "description is required"})
  	case len(r.Description) < 20:
  		errs = append(errs, FieldError{"description", "description must be at least 20 characters"})
  	case len(r.Description) > 5000:
  		errs = append(errs, FieldError{"description", "description must not exceed 5000 characters"})
  	}

  	// Price
  	switch {
  	case r.Price < 0:
  		errs = append(errs, FieldError{"price", "price must be a positive number"})
  	case r.Price > 999_999_999:
  		errs = append(errs, FieldError{"price", "price must not exceed 999,999,999"})
  	}

  	// Category
  	if r.CategoryID <= 0 {
  		errs = append(errs, FieldError{"category_id", "a valid category is required"})
  	}

  	// Condition (optional but constrained)
  	if r.Condition != "" {
  		valid := map[string]bool{"new": true, "used": true, "refurbished": true}
  		if !valid[r.Condition] {
  			errs = append(errs, FieldError{"condition",
  				"condition must be one of: new, used, refurbished"})
  		}
  	}

  	// Location
  	errs = append(errs, r.Location.Validate()...)

  	// Images (optional, max 10, valid URLs)
  	if len(r.Images) > 10 {
  		errs = append(errs, FieldError{"images", "maximum 10 images allowed"})
  	} else {
  		for i, img := range r.Images {
  			img = SanitizeString(img)
  			r.Images[i] = img
  			if img != "" && !urlRegex.MatchString(img) {
  				errs = append(errs, FieldError{
  					fmt.Sprintf("images[%d]", i),
  					"must be a valid HTTP/HTTPS URL",
  				})
  			}
  		}
  	}

  	return errs
  }

  // UpdateListingRequest is the payload for PUT /listings/:id.
  type UpdateListingRequest struct {
  	Title       *string   `json:"title"`
  	Description *string   `json:"description"`
  	Price       *float64  `json:"price"`
  	Condition   *string   `json:"condition"`
  	Status      *string   `json:"status"`
  }

  func (r *UpdateListingRequest) Validate() ValidationErrors {
  	var errs ValidationErrors

  	if r.Title != nil {
  		s := SanitizeString(*r.Title)
  		r.Title = &s
  		if len(s) < 5 {
  			errs = append(errs, FieldError{"title", "title must be at least 5 characters"})
  		} else if len(s) > 200 {
  			errs = append(errs, FieldError{"title", "title must not exceed 200 characters"})
  		}
  	}

  	if r.Description != nil {
  		s := SanitizeRichText(*r.Description)
  		r.Description = &s
  		if len(s) < 20 {
  			errs = append(errs, FieldError{"description", "description must be at least 20 characters"})
  		} else if len(s) > 5000 {
  			errs = append(errs, FieldError{"description", "description must not exceed 5000 characters"})
  		}
  	}

  	if r.Price != nil && (*r.Price < 0 || *r.Price > 999_999_999) {
  		errs = append(errs, FieldError{"price", "price must be between 0 and 999,999,999"})
  	}

  	if r.Condition != nil {
  		s := SanitizeString(*r.Condition)
  		r.Condition = &s
  		valid := map[string]bool{"new": true, "used": true, "refurbished": true}
  		if !valid[s] {
  			errs = append(errs, FieldError{"condition", "must be one of: new, used, refurbished"})
  		}
  	}

  	if r.Status != nil {
  		s := SanitizeString(*r.Status)
  		r.Status = &s
  		valid := map[string]bool{"active": true, "sold": true, "inactive": true}
  		if !valid[s] {
  			errs = append(errs, FieldError{"status", "must be one of: active, sold, inactive"})
  		}
  	}

  	return errs
  }

  // PlaceBidRequest is the payload for POST /auctions/:id/bid.
  type PlaceBidRequest struct {
  	Amount float64 `json:"amount" binding:"required"`
  }

  func (r *PlaceBidRequest) Validate() ValidationErrors {
  	var errs ValidationErrors
  	if r.Amount <= 0 {
  		errs = append(errs, FieldError{"amount", "bid amount must be greater than 0"})
  	}
  	if r.Amount > 999_999_999 {
  		errs = append(errs, FieldError{"amount", "bid amount must not exceed 999,999,999"})
  	}
  	return errs
  }

  // SendMessageRequest is the payload for POST /chats/:id/messages.
  type SendMessageRequest struct {
  	Content string `json:"content" binding:"required"`
  }

  func (r *SendMessageRequest) Validate() ValidationErrors {
  	var errs ValidationErrors
  	r.Content = SanitizeRichText(r.Content)
  	switch {
  	case r.Content == "":
  		errs = append(errs, FieldError{"content", "message content is required"})
  	case len(r.Content) > 2000:
  		errs = append(errs, FieldError{"content", "message must not exceed 2000 characters"})
  	}
  	return errs
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Shared validators
  // ════════════════════════════════════════════════════════════════════════════

  // ValidatePassword checks password strength rules:
  //   - Minimum 8 characters
  //   - At least one uppercase letter
  //   - At least one lowercase letter
  //   - At least one digit
  func ValidatePassword(field, p string) ValidationErrors {
  	var errs ValidationErrors

  	if len(p) < 8 {
  		errs = append(errs, FieldError{field, "must be at least 8 characters"})
  		return errs // early return: further checks are meaningless on very short passwords
  	}
  	if len(p) > 128 {
  		errs = append(errs, FieldError{field, "must not exceed 128 characters"})
  		return errs
  	}

  	var hasUpper, hasLower, hasDigit bool
  	for _, r := range p {
  		switch {
  		case unicode.IsUpper(r):
  			hasUpper = true
  		case unicode.IsLower(r):
  			hasLower = true
  		case unicode.IsDigit(r):
  			hasDigit = true
  		}
  	}

  	if !hasUpper {
  		errs = append(errs, FieldError{field, "must contain at least one uppercase letter (A-Z)"})
  	}
  	if !hasLower {
  		errs = append(errs, FieldError{field, "must contain at least one lowercase letter (a-z)"})
  	}
  	if !hasDigit {
  		errs = append(errs, FieldError{field, "must contain at least one number (0-9)"})
  	}

  	return errs
  }

  // ValidateEmail returns nil if the email is well-formed, or a FieldError.
  func ValidateEmail(field, email string) ValidationErrors {
  	var errs ValidationErrors
  	email = strings.TrimSpace(strings.ToLower(email))
  	switch {
  	case email == "":
  		errs = append(errs, FieldError{field, "email is required"})
  	case len(email) > 255:
  		errs = append(errs, FieldError{field, "email must not exceed 255 characters"})
  	case !emailRegex.MatchString(email):
  		errs = append(errs, FieldError{field, "must be a valid email address"})
  	}
  	return errs
  }

  // ValidatePhone returns nil if the phone is in E.164 format or empty (optional).
  func ValidatePhone(field, phone string) ValidationErrors {
  	if phone == "" {
  		return nil // phone is optional
  	}
  	if !e164Regex.MatchString(phone) {
  		return ValidationErrors{FieldError{field, "must be in E.164 format (e.g. +971501234567)"}}
  	}
  	return nil
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Sanitization helpers
  // ════════════════════════════════════════════════════════════════════════════

  // SanitizeString removes control characters, strips HTML tags, and trims whitespace.
  // Use for short text fields: names, titles, addresses, etc.
  func SanitizeString(s string) string {
  	s = controlCharRegex.ReplaceAllString(s, "")  // remove control chars
  	s = htmlTagRegex.ReplaceAllString(s, "")       // strip HTML tags
  	s = strings.TrimSpace(s)                       // trim leading/trailing whitespace
  	return s
  }

  // SanitizeName removes control characters, strips HTML, trims space, and
  // normalises multiple internal spaces to a single space.
  func SanitizeName(s string) string {
  	s = SanitizeString(s)
  	// Collapse multiple consecutive spaces into one
  	parts := strings.Fields(s)
  	return strings.Join(parts, " ")
  }

  // SanitizeRichText is for multi-line content (descriptions, messages).
  // Preserves \n and \r\n line endings but removes dangerous control characters
  // and HTML tags.
  func SanitizeRichText(s string) string {
  	s = controlCharRegex.ReplaceAllString(s, "") // strips control chars (keeps \n, \r)
  	s = htmlTagRegex.ReplaceAllString(s, "")     // strip HTML tags
  	s = strings.TrimSpace(s)
  	return s
  }

  // SanitizeSearchQuery sanitises a free-text search query:
  // strips HTML, control chars, and collapses extra whitespace.
  func SanitizeSearchQuery(q string) string {
  	q = SanitizeString(q)
  	parts := strings.Fields(q)
  	return strings.Join(parts, " ")
  }
  