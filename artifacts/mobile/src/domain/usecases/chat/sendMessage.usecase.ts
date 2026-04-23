import type { Message, SendMessageInput } from "../../entities";
import type { ChatRepository } from "../../repositories/chat.repository";
import { LIMITS } from "../../../config/constants";
import { ValidationError } from "../../../core/utils/errors";

export class SendMessageUseCase {
  constructor(private readonly chat: ChatRepository) {}

  execute(input: SendMessageInput): Promise<Message> {
    const text = input.text.trim();
    if (!text) {
      throw new ValidationError("Message cannot be empty", { text: "Required" });
    }
    if (text.length > LIMITS.maxMessageLength) {
      throw new ValidationError("Message too long", {
        text: `Max ${LIMITS.maxMessageLength} characters`,
      });
    }
    return this.chat.sendMessage({ ...input, text });
  }
}
