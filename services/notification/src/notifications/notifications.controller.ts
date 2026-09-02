import { Controller, Get, Headers, Param, Patch } from "@nestjs/common";
import { NotificationsService } from "./notifications.service";

@Controller("/notifications")
export class NotificationsController {
  constructor(private readonly svc: NotificationsService) {}

  @Get()
  list(@Headers("x-user-id") userId: string) {
    return this.svc.list(userId);
  }

  @Patch("/:id/read")
  read(@Headers("x-user-id") userId: string, @Param("id") id: string) {
    return this.svc.markRead(userId, id);
  }
}
