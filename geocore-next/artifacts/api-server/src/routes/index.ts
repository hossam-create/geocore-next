import { Router, type IRouter } from "express";
import healthRouter from "./health";
import currenciesRouter from "./currencies";
import locationRouter from "./location";
import validationRouter from "./validation";
import kycRouter from "./kyc";
import aiPricingRouter from "./ai-pricing";
import mediaRouter from "./media";
import authRouter from "./auth";
import aiSearchRouter from "./ai-search";
import aiRecommendRouter from "./ai-recommend";

const router: IRouter = Router();

router.use(healthRouter);
router.use(currenciesRouter);
router.use(locationRouter);
router.use(validationRouter);
router.use(kycRouter);
router.use(aiPricingRouter);
router.use("/v1/media", mediaRouter);
router.use(authRouter);
router.use(aiSearchRouter);
router.use("/v1/ai/recommend", aiRecommendRouter);

export default router;
