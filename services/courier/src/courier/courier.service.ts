import { ForbiddenException, Injectable, Logger, OnModuleInit } from "@nestjs/common";
import { PrismaService } from "../prisma/prisma.service";
import { MqService } from "../mq/mq.service";

const DEMO_COURIER = "33333333-3333-3333-3333-333333333333";

@Injectable()
export class CourierService implements OnModuleInit {
  private readonly log = new Logger(CourierService.name);
  constructor(
    private readonly prisma: PrismaService,
    private readonly mq: MqService,
  ) {}

  async onModuleInit() {
    await this.prisma.courierProfile.upsert({
      where: { userId: DEMO_COURIER },
      update: {},
      create: { userId: DEMO_COURIER, name: "Giorgi K.", isOnline: false },
    });
    await this.mq.consume("courier.order.ready", "order.ready", (body) => this.assign(body));
  }

  async ensure(userId: string, name: string) {
    return this.prisma.courierProfile.upsert({
      where: { userId },
      update: { name },
      create: { userId, name, isOnline: false },
    });
  }

  async setOnline(userId: string, name: string, online: boolean) {
    const profile = await this.ensure(userId, name);
    return this.prisma.courierProfile.update({
      where: { userId: profile.userId },
      data: { isOnline: online },
    });
  }

  async me(userId: string, name: string) {
    const profile = await this.ensure(userId, name);
    const jobs = await this.prisma.delivery.findMany({
      where: { courierId: userId, status: { in: ["assigned", "picked_up"] } },
      orderBy: { assignedAt: "desc" },
    });
    return { ...profile, activeJobs: jobs };
  }

  async jobs(userId: string) {
    return this.prisma.delivery.findMany({
      where: { courierId: userId },
      orderBy: { assignedAt: "desc" },
      take: 50,
    });
  }

  async pickup(userId: string, id: string) {
    const d = await this.prisma.delivery.findUnique({ where: { id } });
    if (!d || d.courierId !== userId) throw new ForbiddenException({ error: { code: "forbidden", message: "not your job" } });
    if (d.status !== "assigned") throw new ForbiddenException({ error: { code: "conflict", message: "cannot pickup" } });
    const updated = await this.prisma.delivery.update({
      where: { id },
      data: { status: "picked_up", pickedUpAt: new Date() },
    });
    await this.mq.publish("courier.picked_up", {
      deliveryId: updated.id,
      orderId: updated.orderId,
      courierId: userId,
      status: "picked_up",
    });
    return updated;
  }

  async deliver(userId: string, id: string) {
    const d = await this.prisma.delivery.findUnique({ where: { id } });
    if (!d || d.courierId !== userId) throw new ForbiddenException({ error: { code: "forbidden", message: "not your job" } });
    if (d.status !== "picked_up") throw new ForbiddenException({ error: { code: "conflict", message: "cannot deliver" } });
    const updated = await this.prisma.delivery.update({
      where: { id },
      data: { status: "delivered", deliveredAt: new Date() },
    });
    await this.mq.publish("courier.delivered", {
      deliveryId: updated.id,
      orderId: updated.orderId,
      courierId: userId,
      status: "delivered",
    });
    return updated;
  }

  private async assign(body: Record<string, unknown>) {
    const orderId = String(body.orderId ?? "");
    if (!orderId) return;
    const existing = await this.prisma.delivery.findUnique({ where: { orderId } });
    if (existing) return;

    const online = await this.prisma.courierProfile.findMany({
      where: { isOnline: true },
      include: { deliveries: { where: { status: { in: ["assigned", "picked_up"] } } } },
    });
    const free = online.filter((c) => c.deliveries.length < 3).sort((a, b) => a.deliveries.length - b.deliveries.length)[0];
    if (!free) {
      this.log.warn(`no courier for ${orderId}`);
      await new Promise((r) => setTimeout(r, 2500));
      throw new Error("no courier online");
    }
    const delivery = await this.prisma.delivery.create({
      data: {
        orderId,
        courierId: free.userId,
        status: "assigned",
        restaurantName: String(body.restaurantName ?? "Restaurant"),
        address: String(body.address ?? ""),
      },
    });
    await this.mq.publish("courier.assigned", {
      deliveryId: delivery.id,
      orderId,
      courierId: free.userId,
      courierName: free.name,
      restaurantName: delivery.restaurantName,
      address: delivery.address,
    });
    this.log.log(`assigned ${orderId} -> ${free.name}`);
  }
}
