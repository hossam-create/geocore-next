#!/usr/bin/env python3
  """
  GeoCore AI Auction Pricing Service
  Inspired by T51-AI-Bidding-and-Auction-Pricing-Agent (DQN bidding concepts)

  Translates reinforcement learning concepts into a production-ready
  statistical pricing engine (no GPU training required).

  Concepts borrowed from the T51 repo:
  - State space: (current_price, time_remaining, bid_count, competition_pressure)
  - Action: bid_increment multiplier
  - Reward model: win_probability * (market_value - price_paid)
  """
  import os
  import math
  import logging
  from flask import Flask, request, jsonify
  from flask_cors import CORS

  logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
  logger = logging.getLogger(__name__)

  app = Flask(__name__)
  CORS(app)

  CATEGORY_MULTIPLIERS = {
      "electronics": 1.15, "vehicles": 1.08, "real_estate": 1.05,
      "fashion": 1.20,     "jewelry": 1.25,  "watches": 1.30,
      "art": 1.18,         "antiques": 1.22, "sports": 1.12,
      "home": 1.10,        "default": 1.12,
  }

  STRATEGIES = {
      "conservative": {"aggression": 0.3, "max_premium": 1.05},
      "balanced":     {"aggression": 0.6, "max_premium": 1.15},
      "aggressive":   {"aggression": 0.9, "max_premium": 1.30},
  }


  def _round_gcc(amount):
      if amount >= 10000: return round(amount / 500) * 500
      if amount >= 1000:  return round(amount / 50) * 50
      if amount >= 100:   return round(amount / 5) * 5
      return round(amount)


  def _sigmoid(x):
      return 1 / (1 + math.exp(-x))


  def predict(d):
      cp   = float(d["current_price"])
      sp   = float(d["starting_price"])
      rp   = float(d["reserve_price"])   if d.get("reserve_price")   else None
      ev   = float(d["estimated_value"]) if d.get("estimated_value") else None
      sr   = int(d["seconds_remaining"])
      ts   = int(d["total_seconds"])
      bc   = int(d["bid_count"])
      cat  = str(d["category"]).lower()
      stgy = str(d.get("strategy", "balanced"))
      cur  = str(d.get("currency", "AED"))

      cm   = CATEGORY_MULTIPLIERS.get(cat, CATEGORY_MULTIPLIERS["default"])
      strat = STRATEGIES.get(stgy, STRATEGIES["balanced"])

      # Market value estimation
      pg = cp / max(sp, 1)
      mv = ev if ev else cp * cm * (1 + math.log(max(pg, 1)) * 0.1)

      # Time urgency [0,1]
      tr = sr / max(ts, 1)
      urg = min((1.0 - tr) * 1.5, 1.0) if tr < 0.1 else 1.0 - tr

      # Competition pressure [0,1]
      bph = (bc / max(ts - sr, 1)) * 3600
      comp = min(bph / 20.0, 1.0)

      # Optimal bid
      incr_pct = 0.02 + urg * 0.03 * strat["aggression"] + comp * 0.02 * strat["aggression"]
      opt_bid  = _round_gcc(cp * (1 + incr_pct))
      max_bid  = _round_gcc(mv * strat["max_premium"])

      # Win probability
      vh = (mv - cp) / max(mv, 1)
      wp = _sigmoid(vh * 3 - urg * 2 + strat["aggression"] * 1.5)

      should = opt_bid <= max_bid and wp > (0.3 - strat["aggression"] * 0.2)
      if rp and cp < rp:
          opt_bid = max(opt_bid, _round_gcc(rp * 1.01))
          should  = True

      parts = []
      if urg > 0.8:   parts.append("auction ending soon")
      elif urg > 0.5: parts.append("mid-auction phase")
      if comp > 0.6:  parts.append("high competition detected")
      parts.append(f"{stgy} strategy applied")
      parts.append(f"{round(wp*100)}% estimated win probability")
      action = "Recommend bidding" if should else "Hold — price near ceiling"

      return {
          "should_bid": should,
          "optimal_bid": round(opt_bid, 2),
          "max_bid": round(max_bid, 2),
          "min_increment": round(cp * 0.01, 2),
          "estimated_market_value": round(mv, 2),
          "win_probability": round(wp, 3),
          "urgency_score": round(urg, 3),
          "competition_score": round(comp, 3),
          "strategy": stgy,
          "explanation": f"{action}. {'; '.join(parts)}.",
          "currency": cur,
      }


  @app.get("/health")
  def health():
      return jsonify({"status": "ok", "service": "geocore-ai-pricing"})


  @app.post("/predict")
  def predict_bid():
      data = request.get_json(force=True, silent=True) or {}
      required = ["current_price", "starting_price", "seconds_remaining", "total_seconds", "bid_count", "category"]
      missing = [k for k in required if k not in data]
      if missing:
          return jsonify({"error": f"Missing: {', '.join(missing)}"}), 400
      try:
          return jsonify({"success": True, "data": predict(data)})
      except Exception as e:
          logger.exception("Prediction error")
          return jsonify({"error": str(e)}), 500


  @app.get("/strategies")
  def list_strategies():
      return jsonify({"strategies": [
          {"id": "conservative", "name": "Conservative", "description": "Low risk, stay well below market value"},
          {"id": "balanced",     "name": "Balanced",     "description": "Optimal risk/reward for most auctions"},
          {"id": "aggressive",   "name": "Aggressive",   "description": "Win at all costs, up to 30% premium"},
      ]})


  @app.get("/categories")
  def list_categories():
      return jsonify({"multipliers": CATEGORY_MULTIPLIERS})


  if __name__ == "__main__":
      port = int(os.environ.get("AI_PRICING_PORT", 8090))
      logger.info(f"GeoCore AI Pricing Service on port {port}")
      app.run(host="0.0.0.0", port=port, debug=os.environ.get("APP_ENV") != "production")
  