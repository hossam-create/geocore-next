package notifications

  import (
        "context"
        "encoding/json"
        "log/slog"
        "strconv"
        "time"

        "github.com/geocore-next/backend/internal/users"
        pkgemail "github.com/geocore-next/backend/pkg/email"
        "github.com/google/uuid"
        "gorm.io/gorm"
  )

  // Service coordinates the multi-channel notification delivery pipeline:
  //   1. Check preferences
  //   2. Save in-app notification (DB)
  //   3. Broadcast via WebSocket (immediate)
  //   4. Queue push via FCM (async)
  //   5. Queue email via Redis (async, consumed by email worker)
  type Service struct {
        db  *gorm.DB
        hub *Hub
        fcm *FCMClient
  }

  // NotifyInput is the payload passed to the delivery pipeline.
  type NotifyInput struct {
        UserID uuid.UUID
        Type   string
        Title  string
        Body   string
        Data   map[string]string // arbitrary key-value pairs included in push + WS
  }

  func NewService(db *gorm.DB, hub *Hub, fcm *FCMClient) *Service {
        return &Service{db: db, hub: hub, fcm: fcm}
  }

  // Notify runs the full delivery pipeline for a single user notification.
  // Designed to be called as `go svc.Notify(input)` from other packages.
  func (s *Service) Notify(input NotifyInput) {
        ctx := context.Background()
        _ = ctx // available for future queue writes

        // ── 1. Load preferences ────────────────────────────────────────────────────
        var prefs NotificationPreference
        if err := s.db.First(&prefs, "user_id = ?", input.UserID).Error; err != nil {
                // No preferences row → use defaults (all enabled)
                prefs = NotificationPreference{InAppEnabled: true}
        }

        // ── 2. Save in-app notification ────────────────────────────────────────────
        if !prefs.InAppEnabled {
                return
        }

        dataJSON := "{}"
        if input.Data != nil {
                if b, err := json.Marshal(input.Data); err == nil {
                        dataJSON = string(b)
                }
        }

        notif := Notification{
                UserID: input.UserID,
                Type:   input.Type,
                Title:  input.Title,
                Body:   input.Body,
                Data:   dataJSON,
        }
        if err := s.db.Create(&notif).Error; err != nil {
                slog.Error("notify: failed to save in-app notification",
                        "user_id", input.UserID.String(), "error", err.Error())
                return
        }

        // ── 3. WebSocket broadcast (immediate) ────────────────────────────────────
        s.hub.BroadcastToUser(input.UserID.String(), &notif)

        // ── 4. FCM push notifications ─────────────────────────────────────────────
        go s.sendPush(input, &prefs)

        // ── 5. Email notifications ────────────────────────────────────────────────
        go s.sendEmail(input, &prefs)
  }

  func (s *Service) sendPush(input NotifyInput, prefs *NotificationPreference) {
        if s.fcm == nil {
                return
        }

        // Check push preference for this notification type
        if !s.shouldSendPush(input.Type, prefs) {
                return
        }

        // Fetch user's push tokens
        var tokens []PushToken
        s.db.Where("user_id = ?", input.UserID).Find(&tokens)
        if len(tokens) == 0 {
                return
        }

        rawTokens := make([]string, len(tokens))
        for i, t := range tokens {
                rawTokens[i] = t.Token
        }

        s.fcm.SendMulticast(rawTokens, input.Title, input.Body, input.Data)
  }

  func (s *Service) sendEmail(input NotifyInput, prefs *NotificationPreference) {
        // Check email preference for this notification type
        if !s.shouldSendEmail(input.Type, prefs) {
                return
        }

        // Look up the user's email address
        var u users.User
        if err := s.db.Select("email, name").First(&u, "id = ?", input.UserID).Error; err != nil {
                return
        }

        switch input.Type {
        case TypeOutbid:
                auctionTitle := input.Data["auction_title"]
                if auctionTitle == "" {
                        auctionTitle = "an auction"
                }
                var newAmount float64
                if a, err := strconv.ParseFloat(input.Data["amount"], 64); err == nil {
                        newAmount = a
                }
                currency := input.Data["currency"]
                _ = pkgemail.SendOutbidEmail(u.Email, u.Name, auctionTitle, newAmount, currency)
        }
        // Other email types (auction won, purchase) are triggered directly by the
        // relevant handlers via pkg/email functions for richer context.
  }

  func (s *Service) shouldSendEmail(notifType string, p *NotificationPreference) bool {
        switch notifType {
        case TypeOutbid:
                return p.EmailOutbid
        case TypeNewMessage:
                return p.EmailMessage
        case TypeNewBid:
                return p.EmailNewBid
        case TypeListingApproved:
                return p.EmailListingApproved
        default:
                return false
        }
  }

  func (s *Service) shouldSendPush(notifType string, p *NotificationPreference) bool {
        switch notifType {
        case TypeNewBid, TypeAuctionWon, TypeAuctionEnded:
                return p.PushNewBid
        case TypeOutbid:
                return p.PushOutbid
        case TypeNewMessage:
                return p.PushMessage
        default:
                return true
        }
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Auto-release escrow after 7 days (background goroutine)
  // ════════════════════════════════════════════════════════════════════════════

  // StartAutoReleaseWorker checks every hour for escrow accounts that have been
  // held for more than 7 days and releases them automatically.
  func StartAutoReleaseWorker(db *gorm.DB, svc *Service) {
        go func() {
                for {
                        time.Sleep(1 * time.Hour)
                        autoReleaseEscrows(db, svc)
                }
        }()
  }

  type escrowReleaseRow struct {
        ID      uuid.UUID
        BuyerID uuid.UUID
        Amount  float64
        Currency string
  }

  func autoReleaseEscrows(db *gorm.DB, svc *Service) {
        cutoff := time.Now().Add(-7 * 24 * time.Hour)
        var rows []escrowReleaseRow

        db.Raw(`
                SELECT ea.id, ea.buyer_id, ea.amount, ea.currency
                FROM escrow_accounts ea
                WHERE ea.status = 'held' AND ea.created_at < ?
                LIMIT 100
        `, cutoff).Scan(&rows)

        for _, r := range rows {
                now := time.Now()
                db.Exec(
                        "UPDATE escrow_accounts SET status='released', released_at=?, notes=? WHERE id=?",
                        now, "Auto-released after 7 days", r.ID,
                )
                svc.Notify(NotifyInput{
                        UserID: r.BuyerID,
                        Type:   TypeEscrowReleased,
                        Title:  "Payment Released",
                        Body:   "Your escrow has been automatically released to the seller.",
                        Data:   map[string]string{"escrow_id": r.ID.String()},
                })
                slog.Info("escrow auto-released", "escrow_id", r.ID.String())
        }
  }
  