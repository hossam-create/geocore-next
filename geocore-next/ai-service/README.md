# GeoCore AI Pricing Service

  Statistical auction pricing microservice inspired by the
  [T51-AI-Bidding-and-Auction-Pricing-Agent](https://github.com/Kavya-Upadhyay/T51-AI-Bidding-and-Auction-Pricing-Agent)
  DQN bidding concepts.

  ## Concept

  Translates T51 DQN reinforcement learning auction model into a
  production-ready statistical engine (no GPU or training required):

  | T51 RL concept | Our implementation |
  |---|---|
  | State space (price, time, bids) | Urgency + competition scores |
  | Action: bid increment multiplier | Optimal bid calculation |
  | Reward: win_prob × profit | Win probability estimate |
  | Strategy exploration vs exploitation | Conservative / Balanced / Aggressive |

  ## API Endpoints

  | Method | Endpoint | Description |
  |--------|----------|-------------|
  | GET | /health | Service health check |
  | POST | /predict | Bid recommendation for one auction |
  | GET | /strategies | List available bid strategies |
  | GET | /categories | GCC category price multipliers |

  ### POST /predict — Request body

  ```json
  {
    "current_price": 5000,
    "starting_price": 1000,
    "reserve_price": 3000,
    "estimated_value": 7000,
    "seconds_remaining": 3600,
    "total_seconds": 86400,
    "bid_count": 15,
    "category": "jewelry",
    "strategy": "balanced",
    "currency": "AED"
  }
  ```

  ### POST /predict — Response

  ```json
  {
    "success": true,
    "data": {
      "should_bid": true,
      "optimal_bid": 5250,
      "max_bid": 8750,
      "min_increment": 50.0,
      "estimated_market_value": 7000,
      "win_probability": 0.712,
      "urgency_score": 0.458,
      "competition_score": 0.234,
      "strategy": "balanced",
      "explanation": "Recommend bidding. mid-auction phase; balanced strategy applied; 71% estimated win probability.",
      "currency": "AED"
    }
  }
  ```

  ## Running

  ```bash
  pip install -r requirements.txt
  python main.py
  # Production: gunicorn main:app -b 0.0.0.0:8090 -w 4
  ```

  ## GCC Currency Rounding

  Bid amounts are rounded to psychologically appealing price points for
  AED/SAR/KWD/QAR/BHD/OMR markets:
  - ≥ 10,000 → nearest 500
  - ≥ 1,000 → nearest 50
  - ≥ 100 → nearest 5
  