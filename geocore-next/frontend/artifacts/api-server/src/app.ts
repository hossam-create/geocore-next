import express, { type Express, type Request, type Response } from "express";
import cors from "cors";
import pinoHttp from "pino-http";
import router from "./routes";
import { logger } from "./lib/logger";

const GO_BACKEND = "https://geo-core-next.replit.app";

const app: Express = express();

app.use(
  pinoHttp({
    logger,
    serializers: {
      req(req) {
        return {
          id: req.id,
          method: req.method,
          url: req.url?.split("?")[0],
        };
      },
      res(res) {
        return {
          statusCode: res.statusCode,
        };
      },
    },
  }),
);
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// ── Local API routes (auth, media, kyc, ai, health, etc.) ───────────────────
app.use("/api", router);

// ── Proxy catch-all → Go backend ─────────────────────────────────────────────
// Any route not matched above gets forwarded to the Go backend.
// This allows the web app to point solely at this api-server.
app.use("/api", async (req: Request, res: Response) => {
  const targetUrl = `${GO_BACKEND}${req.originalUrl}`;
  const authHeader = req.headers.authorization;

  try {
    const init: RequestInit = {
      method: req.method,
      headers: {
        "Content-Type": "application/json",
        ...(authHeader ? { Authorization: authHeader } : {}),
      },
      ...(req.method !== "GET" && req.method !== "HEAD"
        ? { body: JSON.stringify(req.body) }
        : {}),
    };

    const upstream = await fetch(targetUrl, init);
    const text = await upstream.text();

    res.status(upstream.status);
    upstream.headers.forEach((val, key) => {
      if (!["transfer-encoding", "connection"].includes(key.toLowerCase())) {
        res.setHeader(key, val);
      }
    });
    res.send(text);
  } catch (err) {
    logger.warn({ err, targetUrl }, "proxy: upstream request failed");
    res.status(502).json({ success: false, message: "Upstream service unavailable" });
  }
});

export default app;
