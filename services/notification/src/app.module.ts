import { Module } from "@nestjs/common";
import { PrismaService } from "./prisma/prisma.service";
import { MqService } from "./mq/mq.service";
import { MailService } from "./mail/mail.service";
import { HealthController } from "./health/health.controller";
import { NotificationsController } from "./notifications/notifications.controller";
import { NotificationsService } from "./notifications/notifications.service";

@Module({
  controllers: [HealthController, NotificationsController],
  providers: [PrismaService, MqService, MailService, NotificationsService],
})
export class AppModule {}
