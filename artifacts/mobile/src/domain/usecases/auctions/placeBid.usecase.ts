import type {
  Auction,
  Bid,
  PlaceBidInput,
} from "../../entities";
import type { AuctionRepository } from "../../repositories/auction.repository";
import { LIMITS } from "../../../config/constants";
import { ValidationError } from "../../../core/utils/errors";
import { isAuctionLive } from "../../entities/auction.entity";

export interface PlaceBidRequest extends PlaceBidInput {
  readonly auction?: Auction;
}

export class PlaceBidUseCase {
  constructor(private readonly auctions: AuctionRepository) {}

  async execute(req: PlaceBidRequest): Promise<Bid> {
    const auction = req.auction ?? (await this.auctions.get(req.auctionId));

    if (!isAuctionLive(auction)) {
      throw new ValidationError("Auction is not currently accepting bids", {
        auction: "Auction has ended or not started",
      });
    }

    const minNextBid =
      auction.currentBid.amount + auction.minIncrement.amount;

    const fields: Record<string, string> = {};
    if (req.amount <= 0) {
      fields.amount = "Bid amount must be positive";
    } else if (req.amount < minNextBid) {
      fields.amount = `Bid must be at least ${minNextBid} ${auction.currentBid.currency}`;
    }
    if (req.isAuto) {
      if (req.maxAmount === undefined) {
        fields.maxAmount = "Max amount is required for auto-bid";
      } else if (req.maxAmount < req.amount + LIMITS.minAuctionBidStep) {
        fields.maxAmount = "Max amount must be greater than current bid";
      }
    }
    if (Object.keys(fields).length > 0) {
      throw new ValidationError("Invalid bid", fields);
    }

    return this.auctions.placeBid(req);
  }
}
