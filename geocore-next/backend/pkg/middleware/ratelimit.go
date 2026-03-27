package middleware

  import (
        "context"
        "fmt"
        "log/slog"
        "math/rand"
        "net/http"
        "os"
        "strconv"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/redis/go-redis/v9"
  )

  // ════════════════════════════════════════════════════════════════════════════
  // Sliding-window Lua script
  // ════════════════════════════════════════════════════════════════════════════
  //
  // Atomically implements the sliding window rate limiting algorithm.
  //
  // KEYS[1] = Redis sorted-set key for this rate-limit bucket
  // ARGV[1] = current time in milliseconds (string)
  // ARGV[2] = window size in milliseconds  (string)
  // ARGV[3] = request limit                (string)
  // ARGV[4] = unique member ID             (string) — prevents score collisions
  //
  // Returns: { current_count, remaining, reset_unix_seconds }
  //   - remaining   = 0 when the request is rejected
  //   - reset_unix  = 0 when the request is admitted (no cooldown to report)
  var slidingWindowScript = redis.NewScript(`
  local key    = KEYS[1]
  local now    = tonumber(ARGV[1])
  local window = tonumber(ARGV[2])
  local limit  = tonumber(ARGV[3])
  local uid    = ARGV[4]

  redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

  local count = tonumber(redis.call('ZCARD', key))

  if count < limit then
      redis.call('ZADD', key, now, uid)
      redis.call('PEXPIRE', key, window + 1000)
      return {count + 1, limit - count - 1, 0}
  else
      local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
      local reset_ms  = tonumber(oldest[2]) + window
      local reset_sec = math.ceil(reset_ms / 1000)
      return {count, 0, reset_sec}
  end
  `)

  // ════════════════════════════════════════════════════════════════════════════
  // RateLimiter
  // ════════════════════════════════════════════════════════════════════════════

  // RateLimiter holds the Redis client and the IP whitelist for a single service.
  type RateLimiter struct {
        rdb       *redis.Client
        whitelist map[string]bool
  }

  // NewRateLimiter constructs a RateLimiter backed by the given Redis client.
  // Whitelisted IPs are read from the RATE_LIMIT_WHITELIST environment variable
  // as a comma-separated list (e.g. "127.0.0.1,10.0.0.1").
  // Requests from whitelisted IPs always pass without consuming a token.
  func NewRateLimiter(rdb *redis.Client) *RateLimiter {
        wl := map[string]bool{
                "127.0.0.1": true, // always allow localhost
                "::1":       true,
        }
        if envWL := os.Getenv("RATE_LIMIT_WHITELIST"); envWL != "" {
                for _, ip := range strings.Split(envWL, ",") {
                        ip = strings.TrimSpace(ip)
                        if ip != "" {
                                wl[ip] = true
                        }
                }
        }
        return &RateLimiter{rdb: rdb, whitelist: wl}
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Middleware factories
  // ════════════════════════════════════════════════════════════════════════════

  // Limit returns a Gin middleware that allows at most limit requests per window.
  // The bucket key is: "ratelimit:{prefix}:{clientIP}".
  //
  // Use this for public endpoints (no auth required).
  //
  // Example:
  //
  //    rl.Limit(10, 15*time.Minute, "auth:login:ip")
  func (rl *RateLimiter) Limit(limit int, window time.Duration, prefix string) gin.HandlerFunc {
        return rl.limitWith(limit, window, func(c *gin.Context) string {
                return fmt.Sprintf("ratelimit:%s:%s", prefix, c.ClientIP())
        })
  }

  // LimitByUser returns a Gin middleware that allows at most limit requests per
  // window, keyed on the authenticated user ID.
  //
  // Must be placed AFTER middleware.Auth() so that "user_id" is in the context.
  //
  // Example:
  //
  //    auth.Use(middleware.Auth())
  //    auth.POST("/resend-verification",
  //        rl.LimitByUser(5, time.Hour, "auth:resend:user"),
  //        h.ResendVerification,
  //    )
  func (rl *RateLimiter) LimitByUser(limit int, window time.Duration, prefix string) gin.HandlerFunc {
        return rl.limitWith(limit, window, func(c *gin.Context) string {
                userID, _ := c.Get("user_id")
                return fmt.Sprintf("ratelimit:%s:%v", prefix, userID)
        })
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Core implementation
  // ════════════════════════════════════════════════════════════════════════════

  // limitWith is the internal factory that builds the middleware using the
  // provided keyFn to derive the Redis bucket key from the request context.
  func (rl *RateLimiter) limitWith(
        limit int,
        window time.Duration,
        keyFn func(*gin.Context) string,
  ) gin.HandlerFunc {
        return func(c *gin.Context) {
                // ── Bypass: whitelisted IPs ──────────────────────────────────────────
                if rl.whitelist[c.ClientIP()] {
                        c.Next()
                        return
                }

                // ── Bypass: health / readiness endpoints ─────────────────────────────
                path := c.Request.URL.Path
                if path == "/health" || path == "/ready" || path == "/metrics" {
                        c.Next()
                        return
                }

                // ── No Redis: fail open ──────────────────────────────────────────────
                if rl.rdb == nil {
                        c.Next()
                        return
                }

                // ── Evaluate sliding window ───────────────────────────────────────────
                key       := keyFn(c)
                nowMs     := time.Now().UnixMilli()
                windowMs  := window.Milliseconds()
                uniqID    := fmt.Sprintf("%d-%d", nowMs, rand.Int63()) //nolint:gosec

                ctx := context.Background()
                res, err := slidingWindowScript.Run(
                        ctx, rl.rdb, []string{key},
                        strconv.FormatInt(nowMs, 10),
                        strconv.FormatInt(windowMs, 10),
                        strconv.Itoa(limit),
                        uniqID,
                ).Int64Slice()

                if err != nil {
                        // Redis error: fail open (allow the request) to avoid service disruption.
                        slog.Error("ratelimit: Redis script error",
                                "key",   key,
                                "error", err.Error(),
                        )
                        c.Next()
                        return
                }

                current   := int(res[0])
                remaining := int(res[1])
                resetUnix := res[2]  // 0 = request was admitted

                // ── Set rate-limit response headers ──────────────────────────────────
                c.Header("X-RateLimit-Limit",     strconv.Itoa(limit))
                c.Header("X-RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))

                if resetUnix > 0 {
                        c.Header("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))
                } else {
                        c.Header("X-RateLimit-Reset", strconv.FormatInt(
                                time.Now().Add(window).Unix(), 10,
                        ))
                }

                // ── Rejected ──────────────────────────────────────────────────────────
                if remaining == 0 && current >= limit {
                        retryAfter := int64(0)
                        if resetUnix > 0 {
                                retryAfter = resetUnix - time.Now().Unix()
                                if retryAfter < 0 {
                                        retryAfter = 0
                                }
                        }

                        // Audit log: rate limit violation
                        slog.Warn("ratelimit: request rejected",
                                "key",         key,
                                "ip",          c.ClientIP(),
                                "path",        path,
                                "limit",       limit,
                                "window_sec",  int(window.Seconds()),
                                "retry_after", retryAfter,
                        )

                        c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
                        c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                                "error":       "rate_limit_exceeded",
                                "message":     fmt.Sprintf("Too many requests. Please try again in %d seconds.", retryAfter),
                                "retry_after": retryAfter,
                        })
                        return
                }

                c.Next()
        }
  }

  // max returns the larger of a or b (Go 1.21+ has built-in max but this ensures compat).
  func max(a, b int) int {
        if a > b {
                return a
        }
        return b
  }
  