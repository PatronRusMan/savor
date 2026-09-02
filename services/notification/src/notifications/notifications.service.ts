import { Injectable, Logger, OnModuleInit } from "@nestjs/common";
import { PrismaService } from "../prisma/prisma.service";
import { MqService } from "../mq/mq.service";
import { MailService } from "../mail/mail.service";

const EMAILS: Record<string, string> = {
  "11111111-1111-1111-1111-111111111111": "customer@savor.dev",
  "22222222-2222-2222-2222-222222222222": "restaurant@savor.dev",
  "33333333-3333-3333-3333-333333333333": "courier@savor.dev",
  "44444444-4444-4444-4444-444444444444": "kitchen@savor.dev",
};

@Injectable()
export class NotificationsService implements OnModuleInit {
  private readonly log = new Logger(NotificationsService.name);

  constructor(
    private readonly prisma: PrismaService,
    private readonly mq: MqService,
    private readonly mail: MailService,
  ) {}

  async onModuleInit() {
    await this.mq.consume("notification.events", "#", (rk, body) => this.onEvent(rk, body));
  }

  async list(userId: string) {
    return this.prisma.notification.findMany({
      where: { userId },
      orderBy: { createdAt: "desc" },
      take: 50,
    });
  }

  async markRead(userId: string, id: string) {
    const n = await this.prisma.notification.findUnique({ where: { id } });
    if (!n || n.userId !== userId) return n;
    return this.prisma.notification.update({ where: { id }, data: { read: true } });
  }

  private async onEvent(rk: string, body: Record<string, unknown>) {
    const orderId = String(body.orderId ?? "");
    const restaurant = String(body.restaurantName ?? "a restaurant");
    const targets = this.targets(rk, body);
    for (const t of targets) {
      await this.prisma.notification.create({
        data: {
          userId: t.userId,
          title: t.title,
          body: t.body,
          kind: rk,
          orderId: orderId || null,
        },
      });
      await this.mail.send(t.email, t.title, `${t.body}\n\nOrder ${orderId} · ${restaurant}`);
    }
    this.log.log(`event ${rk} -> ${targets.length} recipients`);
  }

  private targets(rk: string, body: Record<string, unknown>) {
    const customerId = String(body.customerId ?? "");
    const ownerId = String(body.restaurantOwnerId ?? "");
    const courierId = String(body.courierId ?? "");
    const restaurant = String(body.restaurantName ?? "Restaurant");
    const out: { userId: string; email: string; title: string; body: string }[] = [];
    const add = (userId: string, title: string, text: string) => {
      if (!userId) return;
      out.push({ userId, email: EMAILS[userId] ?? `${userId}@savor.dev`, title, body: text });
    };
    switch (rk) {
      case "order.paid":
        add(ownerId, "New paid order", `${restaurant} has a new paid order.`);
        add(customerId, "Payment confirmed", `Your order from ${restaurant} is in the kitchen queue.`);
        break;
      case "order.accepted":
        add(customerId, "Restaurant accepted", `${restaurant} accepted your order.`);
        break;
      case "order.cooking":
        add(customerId, "Cooking", `${restaurant} started cooking.`);
        break;
      case "order.ready":
        add(customerId, "Ready for pickup", `${restaurant} marked the order ready.`);
        add(ownerId, "Waiting for courier", "A courier is being assigned.");
        break;
      case "courier.assigned":
      case "order.assigned":
        add(customerId, "Courier assigned", "A courier is heading to the restaurant.");
        add(courierId, "New delivery", `Pickup at ${restaurant}.`);
        add(ownerId, "Courier assigned", "A courier was assigned to the order.");
        break;
      case "courier.picked_up":
      case "order.picked_up":
        add(customerId, "On the way", "Your order is on the bike.");
        break;
      case "courier.delivered":
      case "order.delivered":
        add(customerId, "Delivered", "Enjoy the food. Rate it if it was good.");
        add(ownerId, "Order delivered", "The courier completed the drop-off.");
        break;
      case "order.cancelled":
      case "order.payment_failed":
        add(customerId, "Order did not go through", "Payment failed or the order was cancelled.");
        add(ownerId, "Order cancelled", "An order was cancelled.");
        break;
      default:
        break;
    }
    return out;
  }
}
