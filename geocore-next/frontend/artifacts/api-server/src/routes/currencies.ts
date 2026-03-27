import { Router } from "express";

const router = Router();

router.get("/currencies", async (req, res) => {
  try {
    const r = await fetch("https://api.frankfurter.app/currencies");
    if (!r.ok) {
      res.status(502).json({ error: "Upstream currency API unavailable" });
      return;
    }
    const data = await r.json();
    res.json({ data });
  } catch (err) {
    res.status(500).json({ error: "Failed to fetch currencies" });
  }
});

router.get("/currencies/convert", async (req, res) => {
  const { amount, from, to } = req.query as Record<string, string>;
  if (!amount || !from || !to) {
    res.status(400).json({ error: "amount, from, and to are required" });
    return;
  }
  try {
    const r = await fetch(
      `https://api.frankfurter.app/latest?amount=${amount}&from=${from}&to=${to}`
    );
    if (!r.ok) {
      res.status(502).json({ error: "Currency conversion failed" });
      return;
    }
    const data = await r.json();
    res.json({ data });
  } catch {
    res.status(500).json({ error: "Conversion request failed" });
  }
});

export default router;
