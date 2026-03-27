import { Router } from "express";

const router = Router();

router.get("/detect-location", async (req, res) => {
  const ip =
    (req.headers["x-forwarded-for"] as string)?.split(",")[0].trim() ||
    req.socket.remoteAddress ||
    "127.0.0.1";

  if (ip === "127.0.0.1" || ip === "::1" || ip === "::ffff:127.0.0.1") {
    res.json({
      data: {
        country: "United Arab Emirates",
        countryCode: "AE",
        city: "Dubai",
        lat: 25.2048,
        lon: 55.2708,
        timezone: "Asia/Dubai",
      },
    });
    return;
  }

  try {
    const r = await fetch(
      `http://ip-api.com/json/${ip}?fields=country,countryCode,city,lat,lon,timezone,status`
    );
    if (!r.ok) {
      res.status(502).json({ error: "IP geo service unavailable" });
      return;
    }
    const data = await r.json();
    if (data.status === "fail") {
      res.json({ data: { country: "", countryCode: "", city: "", lat: 0, lon: 0, timezone: "" } });
      return;
    }
    res.json({ data });
  } catch {
    res.status(500).json({ error: "Location detection failed" });
  }
});

router.get("/geocode", async (req, res) => {
  const { q } = req.query as { q: string };
  if (!q) {
    res.status(400).json({ error: "q (query) parameter is required" });
    return;
  }
  try {
    const encoded = encodeURIComponent(q);
    const r = await fetch(
      `https://nominatim.openstreetmap.org/search?q=${encoded}&format=json&limit=1`,
      {
        headers: {
          "User-Agent": "GeoCore-Next/1.0 (geocore@example.com)",
          "Accept-Language": "en",
        },
      }
    );
    if (!r.ok) {
      res.status(502).json({ error: "Geocoding service unavailable" });
      return;
    }
    const results = await r.json();
    if (!results.length) {
      res.json({ data: null });
      return;
    }
    res.json({
      data: {
        lat: results[0].lat,
        lon: results[0].lon,
        displayName: results[0].display_name,
      },
    });
  } catch {
    res.status(500).json({ error: "Geocoding failed" });
  }
});

export default router;
