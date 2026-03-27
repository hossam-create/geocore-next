import { Router } from "express";

const router = Router();

router.get("/validate/email", async (req, res) => {
  const { email } = req.query as { email: string };
  if (!email) {
    res.status(400).json({ error: "email parameter required" });
    return;
  }
  const basicRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!basicRegex.test(email)) {
    res.json({ data: { valid: false, disposable: false, domain: "" } });
    return;
  }
  try {
    const r = await fetch(
      `https://eva.pingutil.com/email?email=${encodeURIComponent(email)}`,
      { signal: AbortSignal.timeout(4000) }
    );
    if (!r.ok) {
      res.json({ data: { valid: true, disposable: false, domain: "" } });
      return;
    }
    const body = await r.json();
    res.json({ data: body.data });
  } catch {
    res.json({ data: { valid: true, disposable: false, domain: "" } });
  }
});

router.get("/validate/phone", (req, res) => {
  const { phone } = req.query as { phone: string };
  if (!phone) {
    res.json({ data: { valid: true } });
    return;
  }
  const e164 = /^\+[1-9]\d{7,14}$/;
  res.json({ data: { valid: e164.test(phone) } });
});

export default router;
