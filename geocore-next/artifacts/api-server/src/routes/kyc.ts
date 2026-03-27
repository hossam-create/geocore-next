import { Router } from "express";

const router = Router();

router.get("/v1/kyc/status", (_req, res) => {
  res.json({ status: "not_started" });
});

router.post("/v1/kyc/submit", (_req, res) => {
  res.json({ message: "KYC submission received" });
});

export default router;
