import { Router } from "express";

const router = Router();

router.post("/v1/ai/pricing", (req, res) => {
  const { category, condition, description } = req.body ?? {};
  res.json({
    suggested_price: null,
    price_range: { min: null, max: null },
    currency: "AED",
    message: "AI pricing not configured",
    inputs: { category, condition, description },
  });
});

export default router;
