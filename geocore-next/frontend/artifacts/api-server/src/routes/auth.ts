import { Router, type IRouter, type Request, type Response } from "express";
import crypto from "crypto";

const router: IRouter = Router();

// ── Mock user store (in-memory for development) ──────────────────────────────
interface MockUser {
  id: string;
  name: string;
  email: string;
  phone: string;
  password_hash: string;
  balance: number;
  rating: number;
  isVerified: boolean;
  created_at: string;
}

const mockUsers: MockUser[] = [
  {
    id: "usr_demo_001",
    name: "Ahmed Al-Rashid",
    email: "demo@geocore.com",
    phone: "+971501234567",
    password_hash: crypto.createHash("sha256").update("demo1234").digest("hex"),
    balance: 5000,
    rating: 4.8,
    isVerified: true,
    created_at: new Date(Date.now() - 90 * 24 * 3600 * 1000).toISOString(),
  },
  {
    id: "usr_demo_002",
    name: "Sara Mohammed",
    email: "seller@geocore.com",
    phone: "+966501234567",
    password_hash: crypto.createHash("sha256").update("seller123").digest("hex"),
    balance: 12500,
    rating: 4.6,
    isVerified: true,
    created_at: new Date(Date.now() - 60 * 24 * 3600 * 1000).toISOString(),
  },
  {
    id: "usr_demo_003",
    name: "Test User",
    email: "test@test.com",
    phone: "+97150000000",
    password_hash: crypto.createHash("sha256").update("test123").digest("hex"),
    balance: 1000,
    rating: 4.0,
    isVerified: false,
    created_at: new Date(Date.now() - 10 * 24 * 3600 * 1000).toISOString(),
  },
];

// In-memory token store
const tokenStore = new Map<string, string>(); // token → user_id

function generateToken(): string {
  return `mock_jwt_${crypto.randomBytes(16).toString("hex")}`;
}

function publicUser(u: MockUser) {
  return {
    id: u.id,
    name: u.name,
    email: u.email,
    phone: u.phone,
    balance: u.balance,
    rating: u.rating,
    isVerified: u.isVerified,
    location: "Dubai, UAE",
    created_at: u.created_at,
  };
}

// ── POST /api/v1/auth/login ──────────────────────────────────────────────────
router.post("/v1/auth/login", (req: Request, res: Response) => {
  const { email, password } = req.body;

  if (!email || !password) {
    return res.status(400).json({ success: false, message: "Email and password are required" });
  }

  const user = mockUsers.find((u) => u.email.toLowerCase() === email.toLowerCase());
  if (!user) {
    return res.status(401).json({ success: false, message: "Invalid credentials" });
  }

  const hash = crypto.createHash("sha256").update(password).digest("hex");
  if (hash !== user.password_hash) {
    return res.status(401).json({ success: false, message: "Invalid credentials" });
  }

  const access_token = generateToken();
  const refresh_token = generateToken();
  tokenStore.set(access_token, user.id);
  tokenStore.set(refresh_token, user.id);

  return res.json({
    success: true,
    data: {
      user: publicUser(user),
      access_token,
      refresh_token,
    },
  });
});

// ── POST /api/v1/auth/register ───────────────────────────────────────────────
router.post("/v1/auth/register", (req: Request, res: Response) => {
  const { name, email, phone, password } = req.body;

  if (!name || !email || !password) {
    return res.status(400).json({ success: false, message: "Name, email and password are required" });
  }

  const exists = mockUsers.find((u) => u.email.toLowerCase() === email.toLowerCase());
  if (exists) {
    return res.status(409).json({ success: false, message: "Email already registered" });
  }

  const newUser: MockUser = {
    id: `usr_${crypto.randomBytes(6).toString("hex")}`,
    name,
    email,
    phone: phone || "",
    password_hash: crypto.createHash("sha256").update(password).digest("hex"),
    balance: 0,
    rating: 0,
    isVerified: false,
    created_at: new Date().toISOString(),
  };
  mockUsers.push(newUser);

  const access_token = generateToken();
  const refresh_token = generateToken();
  tokenStore.set(access_token, newUser.id);
  tokenStore.set(refresh_token, newUser.id);

  return res.status(201).json({
    success: true,
    data: {
      user: publicUser(newUser),
      access_token,
      refresh_token,
    },
  });
});

// ── POST /api/v1/auth/refresh ────────────────────────────────────────────────
router.post("/v1/auth/refresh", (req: Request, res: Response) => {
  const { refresh_token } = req.body;
  const userId = tokenStore.get(refresh_token);
  if (!userId) {
    return res.status(401).json({ success: false, message: "Invalid refresh token" });
  }
  const user = mockUsers.find((u) => u.id === userId);
  if (!user) {
    return res.status(401).json({ success: false, message: "User not found" });
  }
  const new_access = generateToken();
  tokenStore.set(new_access, userId);
  return res.json({
    success: true,
    data: { access_token: new_access },
  });
});

// ── GET /api/v1/auth/me ──────────────────────────────────────────────────────
router.get("/v1/auth/me", (req: Request, res: Response) => {
  const authHeader = req.headers.authorization || "";
  const token = authHeader.replace("Bearer ", "");
  const userId = tokenStore.get(token);
  if (!userId) {
    return res.status(401).json({ success: false, message: "Unauthorized" });
  }
  const user = mockUsers.find((u) => u.id === userId);
  if (!user) {
    return res.status(401).json({ success: false, message: "User not found" });
  }
  return res.json({ success: true, data: { user: publicUser(user) } });
});

export default router;
