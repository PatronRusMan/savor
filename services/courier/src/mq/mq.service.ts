import { Injectable, Logger, OnModuleDestroy, OnModuleInit } from "@nestjs/common";
import amqplib from "amqplib";

export const EXCHANGE = "savor.events";

@Injectable()
export class MqService implements OnModuleInit, OnModuleDestroy {
  private readonly log = new Logger(MqService.name);
  private conn!: amqplib.ChannelModel;
  private ch!: amqplib.Channel;
  private boot!: Promise<void>;

  onModuleInit() {
    this.boot = this.connect();
    return this.boot;
  }

  private async connect() {
    const url = process.env.RABBITMQ_URL ?? "amqp://savor:savor@rabbitmq:5672/";
    for (let i = 0; i < 20; i++) {
      try {
        this.conn = await amqplib.connect(url);
        this.ch = await this.conn.createChannel();
        await this.ch.assertExchange(EXCHANGE, "topic", { durable: true });
        this.log.log("rabbitmq connected");
        return;
      } catch (err) {
        this.log.warn(`rabbitmq retry ${i + 1}: ${(err as Error).message}`);
        await new Promise((r) => setTimeout(r, 1000));
      }
    }
    throw new Error("rabbitmq unavailable");
  }

  private async ready() {
    if (!this.boot) this.boot = this.connect();
    await this.boot;
  }

  async publish(routingKey: string, payload: unknown) {
    await this.ready();
    this.ch.publish(EXCHANGE, routingKey, Buffer.from(JSON.stringify(payload)), {
      contentType: "application/json",
      persistent: true,
    });
  }

  async consume(queue: string, key: string, handler: (body: Record<string, unknown>) => Promise<void>) {
    await this.ready();
    await this.ch.assertQueue(queue, { durable: true });
    await this.ch.bindQueue(queue, EXCHANGE, key);
    await this.ch.prefetch(4);
    await this.ch.consume(queue, async (msg) => {
      if (!msg) return;
      try {
        const body = JSON.parse(msg.content.toString()) as Record<string, unknown>;
        await handler(body);
        this.ch.ack(msg);
      } catch (err) {
        this.log.error(`consume ${queue}: ${(err as Error).message}`);
        this.ch.nack(msg, false, true);
      }
    });
  }

  async onModuleDestroy() {
    await this.ch?.close();
    await this.conn?.close();
  }
}
